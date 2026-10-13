export class Semaphore {
  readonly #limit: number;
  #active = 0;
  readonly #waiters: Array<() => void> = [];

  constructor(limit: number) {
    if (!Number.isInteger(limit) || limit < 1) throw new RangeError("limit must be positive");
    this.#limit = limit;
  }

  async use<T>(operation: () => Promise<T>, stage = "queue.semaphore"): Promise<T> {
    await latencyStage(stage, () => this.#acquire());
    try {
      return await operation();
    } finally {
      this.#release();
    }
  }

  async #acquire(): Promise<void> {
    if (this.#active < this.#limit) {
      this.#active++;
      return;
    }
    await new Promise<void>((resolve) => this.#waiters.push(resolve));
    this.#active++;
  }

  #release(): void {
    this.#active--;
    this.#waiters.shift()?.();
  }
}

export class KeyedMutex {
  readonly #tails = new Map<string, Promise<void>>();

  async use<T>(key: string, operation: () => Promise<T>, stage = "queue.mutex"): Promise<T> {
    const previous = this.#tails.get(key) ?? Promise.resolve();
    let release!: () => void;
    const current = new Promise<void>((resolve) => release = resolve);
    const tail = previous.then(() => current);
    this.#tails.set(key, tail);
    await latencyStage(stage, () => previous);
    try {
      return await operation();
    } finally {
      release();
      if (this.#tails.get(key) === tail) this.#tails.delete(key);
    }
  }
}

export class BoundedCache<K, V> {
  readonly #values = new Map<K, { value: V; expiresAt: number }>();

  constructor(readonly capacity: number, readonly ttlMs: number) {}

  get(key: K, now = Date.now()): V | undefined {
    const entry = this.#values.get(key);
    if (!entry) return undefined;
    if (entry.expiresAt <= now) {
      this.#values.delete(key);
      return undefined;
    }
    this.#values.delete(key);
    this.#values.set(key, entry);
    return entry.value;
  }

  set(key: K, value: V, now = Date.now()): void {
    this.#values.delete(key);
    this.#values.set(key, { value, expiresAt: now + this.ttlMs });
    while (this.#values.size > this.capacity) {
      const oldest = this.#values.keys().next().value as K | undefined;
      if (oldest === undefined) break;
      this.#values.delete(oldest);
    }
  }

  get size(): number {
    return this.#values.size;
  }
}

export class SingleFlight<K, V> {
  readonly #pending = new Map<K, Promise<V>>();

  run(key: K, operation: () => Promise<V>): Promise<V> {
    const found = this.#pending.get(key);
    if (found) return found;
    const pending = operation().finally(() => this.#pending.delete(key));
    this.#pending.set(key, pending);
    return pending;
  }
}
import { latencyStage } from "./latency.ts";
