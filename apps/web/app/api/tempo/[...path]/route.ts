import { NextRequest, NextResponse } from "next/server";
import { hasSession } from "@/lib/auth/session";

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  if (!(await hasSession())) return NextResponse.json({ error: "unauthorized" }, { status: 401 });
  const { path } = await context.params; const target = new URL(`/api/v1/${path.join("/")}`, process.env.INTERNAL_API_URL || "http://localhost:8080"); target.search = request.nextUrl.search;
  const response = await fetch(target, { method: request.method, body: ["GET","HEAD"].includes(request.method) ? undefined : await request.text(), headers: { Authorization: `Bearer ${process.env.INTERNAL_API_TOKEN || ""}`, "Content-Type": request.headers.get("Content-Type") || "application/json" }, cache: "no-store" });
  return new NextResponse(response.body, { status: response.status, headers: { "Content-Type": response.headers.get("Content-Type") || "application/json" } });
}
export const GET = proxy; export const POST = proxy; export const PATCH = proxy; export const DELETE = proxy;
