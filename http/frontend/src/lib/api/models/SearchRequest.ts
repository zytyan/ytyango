/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export type SearchRequest = {
	/**
	 * Search keyword
	 */
	q: string;
	/**
	 * Web chat id (stringified int64)
	 */
	ins_id: string;
	page: number;
	limit?: number;
};
