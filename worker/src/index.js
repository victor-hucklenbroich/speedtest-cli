export default {
    async fetch(request) {
        const url = new URL(request.url);
        const cors = { "access-control-allow-origin": "*" };

        // GET
        if (url.pathname === "/down") {
            const bytes = Math.min(parseInt(url.searchParams.get("bytes") || "0", 10) || 0, 1_000_000_000);
            const chunk = new Uint8Array(65536);
            let sent = 0;
            const stream = new ReadableStream({
                pull(controller) {
                    if (sent >= bytes) return controller.close();
                    const n = Math.min(chunk.length, bytes - sent);
                    controller.enqueue(n === chunk.length ? chunk : chunk.subarray(0, n));
                    sent += n;
                },
            });
            return new Response(stream, { headers: { "content-type": "application/octet-stream", ...cors } });
        }

        // POST
        if (url.pathname === "/up" && request.method === "POST") {
            const reader = request.body?.getReader();
            if (reader) while (!(await reader.read()).done) {}
            return new Response("ok", { headers: cors });
        }

        return new Response("not found", { status: 404, headers: cors });
    },
};
