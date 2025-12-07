import { fireEvent, render } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import SearchBar from '../SearchBar.svelte';

describe('SearchBar', () => {
	it('emits submit with the current value', async () => {
		const { component, container, getByRole } = render(SearchBar, {
			props: { value: '测试', loading: false }
		});

		const submitted = vi.fn<(value: string) => void>();
		(component as unknown as { $on: (event: string, handler: (evt: CustomEvent<string>) => void) => void }).$on(
			'submit',
			(event) => submitted(event.detail)
		);

		const input = getByRole('searchbox') as HTMLInputElement;
		await fireEvent.input(input, { target: { value: '新搜索' } });

		const form = container.querySelector('form');
		if (!form) throw new Error('Form not found');

		await fireEvent.submit(form);
		expect(submitted).toHaveBeenCalledWith('新搜索');
	});

	it('shows spinner when loading', () => {
		const { container } = render(SearchBar, { props: { value: '', loading: true } });
		expect(container.querySelector('.spinner')).toBeInTheDocument();
	});
});
