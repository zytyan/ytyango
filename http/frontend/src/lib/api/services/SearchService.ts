/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { SearchRequest } from '../models/SearchRequest';
import type { SearchResult } from '../models/SearchResult';
import type { CancelablePromise } from '../core/CancelablePromise';
import { OpenAPI } from '../core/OpenAPI';
import { request as __request } from '../core/request';
export class SearchService {
    /**
     * Search messages for a group
     * @returns SearchResult Search results
     * @throws ApiError
     */
    public static searchMessages({
        requestBody,
    }: {
        requestBody: SearchRequest,
    }): CancelablePromise<SearchResult> {
        return __request(OpenAPI, {
            method: 'POST',
            url: '/search',
            body: requestBody,
            mediaType: 'application/json',
            errors: {
                400: `Invalid request`,
                500: `Internal server error`,
            },
        });
    }
}
