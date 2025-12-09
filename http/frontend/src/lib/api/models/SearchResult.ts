/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { MeiliMsg } from './MeiliMsg';
export type SearchResult = {
	hits: Array<MeiliMsg>;
	query: string;
	processingTimeMs: number;
	limit: number;
	offset: number;
	estimatedTotalHits: number;
};
