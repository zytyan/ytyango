import type { MaybeInaccessibleMessage } from "grammy/types";

// A regular supergroup reply may have a thread ID without belonging to a forum topic.
export function topicId(message?: MaybeInaccessibleMessage): number {
  return message && "is_topic_message" in message && message.is_topic_message
    ? message.message_thread_id ?? 0
    : 0;
}
