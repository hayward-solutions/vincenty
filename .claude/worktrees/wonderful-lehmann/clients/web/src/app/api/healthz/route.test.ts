import { GET } from "./route";

describe("GET /api/healthz", () => {
  it("returns status ok", async () => {
    const response = GET();
    const json = await response.json();
    expect(json).toEqual({ status: "ok" });
  });
});
