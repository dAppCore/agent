// SPDX-License-Identifier: EUPL-1.2

import { test, expect } from "bun:test"
import { Throttle } from "../src/throttle.ts"

test("Throttle: first send passes, within-window blocked, after-window passes", () => {
  const t = new Throttle(60000)
  expect(t.shouldSend("s", 0)).toBe(true)
  expect(t.shouldSend("s", 30000)).toBe(false)
  expect(t.shouldSend("s", 61000)).toBe(true)
})

test("Throttle: independent per session id", () => {
  const t = new Throttle(60000)
  expect(t.shouldSend("a", 0)).toBe(true)
  expect(t.shouldSend("b", 30000)).toBe(true)
  expect(t.shouldSend("a", 30000)).toBe(false)
})

test("Throttle: baseline advances on each accepted send", () => {
  const t = new Throttle(1000)
  expect(t.shouldSend("s", 0)).toBe(true)
  expect(t.shouldSend("s", 1000)).toBe(true)
  expect(t.shouldSend("s", 1500)).toBe(false)
  expect(t.shouldSend("s", 2000)).toBe(true)
})
