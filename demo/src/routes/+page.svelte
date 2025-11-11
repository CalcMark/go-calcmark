<script>
	import CalcMarkEditor from '$lib/components/CalcMarkEditor.svelte';
</script>

<div class="app">
	<CalcMarkEditor />
</div>

<style>
	.app {
		min-height: 100vh;
	}

	:global(body) {
		margin: 0;
		padding: 0;
		background: #f5f5f5;
	}
</style>
