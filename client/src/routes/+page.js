const server = 'http://localhost:8080/libraries';

let fmApi;

function parseDates(data) {
	data.forEach((item) => {
		if (item.date) item.date = new Date(item.date);
	});
	return data;
}

function init(api) {
	fmApi = api;
	api.intercept('filter-files', (ev) => {
		console.log(ev);
		const { panels, activePanel } = api.getState();
		const id = panels[activePanel].path;
		fetch(
			server + '/files' + (id == '/' ? '' : `/${encodeURIComponent(id)}`) + `?q=${ev?.text || ''}`
		)
			.then((data) => data.json())
			.then((data) => {
				api.exec('set-mode', { mode: ev?.text ? 'search' : 'cards' });
				api.exec('provide-data', {
					id,
					data: parseDates(data)
				});
			});
		return false;
	});
}

function loadData(ev) {
	const id = ev.id;
	fetch(server + '/files/' + encodeURIComponent(id))
		.then((data) => data.json())
		.then((data) => {
			fmApi.exec('provide-data', {
				id,
				data: parseDates(data)
			});
		});
}

export async function load({ fetch }) {
	const res = await fetch(server + '/files');
	if (!res.ok) {
		throw new Error(`Failed to fetch: ${res.status}`);
	}

	const data = await res.json();
	parseDates(data);
	return {
		data,
		init,
		loadData
	};
}
