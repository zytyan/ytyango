const formatter = new Intl.DateTimeFormat('zh-CN', {
	year: 'numeric',
	month: '2-digit',
	day: '2-digit',
	hour: '2-digit',
	minute: '2-digit'
});

export const formatMessageTime = (timestamp: number): string => {
	const ts = Number(timestamp);
	if (Number.isNaN(ts)) return '';
	return formatter.format(new Date(ts));
};

export const ensureMillis = (value?: number | null): number => {
	if (!value) return Date.now();
	return value < 2_000_000_000 ? value * 1000 : value;
};
