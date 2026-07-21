const encoder = new TextEncoder();

function response(result: unknown): Response {
  return new Response(JSON.stringify({ ok: true, result }), {
    headers: { "content-type": "application/json" },
  });
}

Deno.serve({ hostname: "127.0.0.1", port: 18_991 }, async (request) => {
  const method = new URL(request.url).pathname.split("/").at(-1);
  if (method === "getMe") {
    return response({
      id: 99,
      is_bot: true,
      first_name: "Test Bot",
      username: "ytyan_test_bot",
      can_join_groups: true,
      can_read_all_group_messages: true,
      supports_inline_queries: true,
    });
  }
  if (method === "getUpdates") {
    await new Promise((resolve) => setTimeout(resolve, 250));
    return response([]);
  }
  return new Response(encoder.encode(JSON.stringify({ ok: false, description: "unsupported" })), {
    status: 400,
    headers: { "content-type": "application/json" },
  });
});
