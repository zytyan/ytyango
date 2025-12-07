import { fireEvent, render } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import MessageCard from '../MessageCard.svelte';

describe('MessageCard', () => {
	const baseMessage = {
		id: 'demo',
		userId: 10001,
		displayName: 'z',
		username: 'z',
		text: '测试火星车吗，有意思',
		timestamp: Date.now(),
		avatarUrl: null,
		starred: false
	};

	it('renders message content', () => {
		const { getByText, getByLabelText } = render(MessageCard, { props: { message: baseMessage } });
		expect(getByText('z')).toBeInTheDocument();
		expect(getByText('测试火星车吗，有意思')).toBeInTheDocument();
		expect(getByLabelText(/更多操作/)).toBeInTheDocument();
	});

	it('emits toggleStar with id', async () => {
		const { component, getByLabelText } = render(MessageCard, { props: { message: baseMessage } });
		const toggled = vi.fn<(value: string) => void>();
		(component as unknown as { $on: (event: string, handler: (evt: CustomEvent<string>) => void) => void }).$on(
			'toggleStar',
			(event) => toggled(event.detail)
		);

		const starButton = getByLabelText('添加星标');
		await fireEvent.click(starButton);

		expect(toggled).toHaveBeenCalledWith('demo');
	});
});
