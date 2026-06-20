import { describe, it, expect } from "vitest";

describe("vitest setup", () => {
  it("runs a basic assertion", () => {
    expect(1 + 1).toBe(2);
  });

  it("supports describe/it blocks", () => {
    expect(true).toBe(true);
  });
});