import type { MeiliMsg } from '$lib/api';

export const sampleHits: MeiliMsg[] = [
	{
		mongo_id: 'demo-1',
		peer_id: 1,
		from_id: 10001,
		msg_id: 1,
		date: 1_765_023_000_000,
		message: '测试一下多模态输入 @ytyan_bot'
	},
	{
		mongo_id: 'demo-2',
		peer_id: 1,
		from_id: 10001,
		msg_id: 2,
		date: 1_763_625_420_000,
		message: '测试火星车吗，有意思'
	},
	{
		mongo_id: 'demo-3',
		peer_id: 1,
		from_id: 10001,
		msg_id: 3,
		date: 1_763_483_400_000,
		message: '测试新的下载工具'
	}
];

export const sampleUsers: Record<number, { name: string; username?: string | null }> = {
	10001: { name: 'z', username: 'z' }
};
