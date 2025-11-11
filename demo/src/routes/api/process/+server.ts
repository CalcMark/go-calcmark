import { json } from '@sveltejs/kit';
import { processCalcMark } from '$lib/server/calcmark';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ request }) => {
	try {
		const { input } = await request.json();

		if (typeof input !== 'string') {
			return json({ error: 'Input must be a string' }, { status: 400 });
		}

		const result = await processCalcMark(input);
		return json(result);
	} catch (error) {
		console.error('CalcMark processing error:', error);
		return json(
			{ error: error instanceof Error ? error.message : 'Unknown error' },
			{ status: 500 }
		);
	}
};
