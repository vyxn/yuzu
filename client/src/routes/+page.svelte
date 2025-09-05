<script lang="ts">
	import { Filemanager, Willow } from "wx-svelte-filemanager";

	let { data } = $props();
	const drive = {
		used: 15200000000,
		total: 50000000000,
	};
</script>

<Willow>
	<Filemanager init={data.init} data={data.data} {drive} onrequestdata={data.loadData} />
</Willow>
