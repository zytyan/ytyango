import { assertEquals } from "@std/assert";
import { BoundedCache, Semaphore } from "../../src/shared/concurrency.ts";

Deno.test("bounded cache evicts oldest entries", () => {
  const cache = new BoundedCache<string, number>(2, 1_000);
  cache.set("a", 1, 0);
  cache.set("b", 2, 0);
  cache.set("c", 3, 0);
  assertEquals(cache.get("a", 1), undefined);
  assertEquals(cache.size, 2);
});

Deno.test("semaphore enforces its hard concurrency limit", async () => {
  const semaphore = new Semaphore(2);
  let active = 0;
  let maximum = 0;
  await Promise.all(Array.from({ length: 8 }, () =>
    semaphore.use(async () => {
      active++;
      maximum = Math.max(maximum, active);
      await new Promise((resolve) => setTimeout(resolve, 2));
      active--;
    })));
  assertEquals(maximum, 2);
});
