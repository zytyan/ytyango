const STAR_KEY = 'yt-starred-messages';

export const loadStarred = (): Set<string> => {
	if (typeof localStorage === 'undefined') return new Set();
	try {
		const raw = localStorage.getItem(STAR_KEY);
		if (!raw) return new Set();
		const parsed = JSON.parse(raw);
		return new Set(Array.isArray(parsed) ? parsed : []);
	} catch (error) {
		console.error('Failed to parse starred items', error);
		return new Set();
	}
};

export const persistStarred = (ids: Set<string>) => {
	if (typeof localStorage === 'undefined') return;
	localStorage.setItem(STAR_KEY, JSON.stringify([...ids]));
};
