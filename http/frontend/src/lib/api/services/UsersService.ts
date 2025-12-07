/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { UserInfoRequest } from '../models/UserInfoRequest';
import type { UserInfoResponse } from '../models/UserInfoResponse';
import type { CancelablePromise } from '../core/CancelablePromise';
import { OpenAPI } from '../core/OpenAPI';
import { request as __request } from '../core/request';
export class UsersService {
    /**
     * Get basic info for multiple users
     * @returns UserInfoResponse Batch user info
     * @throws ApiError
     */
    public static getUsersInfo({
        requestBody,
    }: {
        requestBody: UserInfoRequest,
    }): CancelablePromise<UserInfoResponse> {
        return __request(OpenAPI, {
            method: 'POST',
            url: '/users/info',
            body: requestBody,
            mediaType: 'application/json',
            errors: {
                400: `Invalid request`,
                500: `Internal server error`,
            },
        });
    }
    /**
     * Get a user's avatar (webp)
     * @returns binary Avatar image in webp format
     * @throws ApiError
     */
    public static getUserAvatar({
        userId,
        tgauth,
    }: {
        /**
         * Numeric user id
         */
        userId: number,
        /**
         * Telegram WebApp init data (raw querystring) signed with sha256(botToken); passed verbatim to server-side verification.
         *
         */
        tgauth: string,
    }): CancelablePromise<Blob> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/users/{userId}/avatar',
            path: {
                'userId': userId,
            },
            query: {
                'tgauth': tgauth,
            },
            errors: {
                401: `Missing or invalid authentication`,
                403: `Authentication failed`,
                404: `Resource not found`,
                500: `Internal server error`,
            },
        });
    }
}
