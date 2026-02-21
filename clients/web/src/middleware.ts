import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

export function middleware(request: NextRequest) {
  const apiUrl = process.env.API_INTERNAL_URL || "http://localhost:8080";
  const { pathname, search } = request.nextUrl;

  return NextResponse.rewrite(new URL(`${apiUrl}${pathname}${search}`));
}

export const config = {
  matcher: "/api/:path*",
};
