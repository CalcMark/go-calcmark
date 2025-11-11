/**
 * Server-side CalcMark WASM loader
 * Loads and initializes the WASM module once on server startup
 */

import { readFileSync } from 'fs';
import { join } from 'path';

interface CalcMarkAPI {
	tokenize(source: string): { tokens: string; error: string | null };
	evaluate(source: string, useGlobalContext: boolean): { results: string; error: string | null };
	evaluateDocument(source: string, useGlobalContext: boolean): { results: string; error: string | null };
	validate(source: string): { diagnostics: string; error: string | null };
	classifyLines(lines: string[]): { classifications: string; error: string | null };
	resetContext(): void;
	getVersion(): string;
}

declare global {
	// eslint-disable-next-line no-var
	var calcmark: CalcMarkAPI | undefined;
	// eslint-disable-next-line no-var
	var Go: {
		new (): {
			importObject: WebAssembly.Imports;
			run(instance: WebAssembly.Instance): Promise<void>;
		};
	};
}

let wasmInitialized = false;
let wasmInitPromise: Promise<void> | null = null;

/**
 * Initialize the CalcMark WASM module
 * Only runs once, subsequent calls return immediately
 */
export async function initCalcMark(): Promise<void> {
	if (wasmInitialized) return;
	if (wasmInitPromise) return wasmInitPromise;

	wasmInitPromise = (async () => {
		try {
			// Load wasm_exec.js (Go's WASM runtime)
			const wasmExecPath = join(process.cwd(), 'static', 'wasm_exec.js');
			await import(wasmExecPath);

			// Load WASM file
			const wasmPath = join(process.cwd(), 'static', 'calcmark.wasm');
			const wasmBuffer = readFileSync(wasmPath);

			// Instantiate WASM
			const go = new global.Go();
			const { instance } = await WebAssembly.instantiate(wasmBuffer, go.importObject);

			// Run Go program (sets up global.calcmark)
			go.run(instance);

			// Wait for initialization
			await new Promise((resolve) => setTimeout(resolve, 100));

			if (!global.calcmark) {
				throw new Error('CalcMark API not initialized');
			}

			wasmInitialized = true;
			console.log('✓ CalcMark WASM initialized on server');
		} catch (error) {
			wasmInitPromise = null;
			throw error;
		}
	})();

	return wasmInitPromise;
}

/**
 * Get the CalcMark API (ensures WASM is initialized)
 */
export async function getCalcMark(): Promise<CalcMarkAPI> {
	await initCalcMark();
	if (!global.calcmark) {
		throw new Error('CalcMark API not available');
	}

	// Debug: log available functions
	console.log('[getCalcMark] Available functions:', Object.keys(global.calcmark));

	return global.calcmark;
}

/**
 * Process CalcMark input and return all results
 */
export async function processCalcMark(input: string) {
	console.log('[processCalcMark] START - input length:', input.length);
	const api = await getCalcMark();

	const lines = input.split('\n');
	console.log('[processCalcMark] Split into', lines.length, 'lines');

	// Step 1: Classify lines
	const classifyResult = api.classifyLines(lines);
	const classifications = classifyResult.error
		? []
		: JSON.parse(classifyResult.classifications);

	// Step 2: Tokenize calculation lines only
	// Returns tokens grouped by line number (1-indexed)
	const tokensByLine: Record<number, any[]> = {};

	for (let i = 0; i < lines.length; i++) {
		const classification = classifications[i];
		const lineNumber = i + 1;

		if (classification && classification.lineType === 'CALCULATION') {
			const tokenResult = api.tokenize(lines[i]);
			if (!tokenResult.error && tokenResult.tokens) {
				tokensByLine[lineNumber] = JSON.parse(tokenResult.tokens);
			}
		}
	}

	// Step 3: Evaluate (single pass for entire document using evaluateDocument)
	// This function handles markdown + calculations properly by classifying first
	const evalResult = api.evaluateDocument(input, true);
	if (evalResult.error) {
		console.log('[processCalcMark] EVALUATION ERROR:', evalResult.error);
	}
	const evaluationResults = evalResult.error ? [] : JSON.parse(evalResult.results);
	console.log('[processCalcMark] Evaluation results count:', evaluationResults.length);

	// Map evaluation results to line numbers and build variable context
	// evaluateDocument already provides OriginalLine, so we just need to build variableContext
	const resultsByLine = evaluationResults; // Already has OriginalLine from evaluateDocument
	const variableContext: Record<string, any> = {}; // varName -> {Value, Symbol, SourceFormat}

	console.log('[processCalcMark] Building variableContext from', evaluationResults.length, 'results');
	for (const result of evaluationResults) {
		const lineNumber = result.OriginalLine;
		const tokens = tokensByLine[lineNumber] || [];
		const assignToken = tokens.find((t: any) => t.type === 'ASSIGN');
		if (assignToken) {
			// First token before ASSIGN is the variable name
			const varToken = tokens.find((t: any) => t.type === 'IDENTIFIER' && t.start < assignToken.start);
			if (varToken) {
				console.log('Adding to variableContext:', varToken.value, '=', result);
				variableContext[varToken.value] = result;
			} else {
				console.log('No IDENTIFIER token before ASSIGN on line', lineNumber, 'tokens:', tokens.map((t: any) => `${t.type}:${t.value}`));
			}
		}
	}
	console.log('[processCalcMark] Final variableContext has', Object.keys(variableContext).length, 'variables');

	// Step 4: Validate
	const validateResult = api.validate(input);
	const diagnostics = validateResult.error ? {} : JSON.parse(validateResult.diagnostics);

	const response = {
		classifications,
		tokensByLine,
		evaluationResults: resultsByLine,
		diagnostics,
		variableContext
	};
	console.log('[processCalcMark] Returning variableContext:', JSON.stringify(variableContext), 'Keys:', Object.keys(variableContext));
	return response;
}
