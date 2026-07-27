// SPDX-License-Identifier: EUPL-1.2

import { test, expect } from "bun:test"
import { Throttle } from "../src/throttle.ts"

test("Throttle: first call always passes", () => {
  const t = new Throttle(60000)
  expect(t.shouldSend("session-1", 0)).toBe(true)
})

test("Throttle: second call within interval fails", () => {
  const t = new Throttle(60000)
  t.shouldSend("session-1", 0) // first call
  expect(t.shouldSend("session-1", 30000)).toBe(false) // within interval
})

test("Throttle: second call after interval passes", () => {
  const t = new Throttle(60000)
  t.shouldSend("session-1", 0) // first call
  expect(t.shouldSend("session-1", 61000)).toBe(true) // after interval
})

test("Throttle: different sessions are independent", () => {
  const t = new Throttle(60000)
  t.shouldSend("session-1", 0) // first call for session-1
  expect(t.shouldSend("session-2", 0)).toBe(true) // first call for session-2
  expect(t.shouldSend("session-1", 30000)).toBe(false) // session-1 within interval
  expect(t.shouldSend("session-2", 30000)).toBe(false) // session-2 within interval
})

test("Throttle: clear removes all timestamps", () => {
  const t = new Throttle(60000)
  t.shouldSend("session-1", 0)
  t.shouldSend("session-2", 0)
  t.clear()
  expect(t.shouldSend("session-1", 1000)).toBe(true)
  expect(t.shouldSend("session-2", 1000)).toBe(true)
})

test("Throttle: clearSession removes specific session timestamp", () => {
  const t = new Throttle(60000)
  t.shouldSend("session-1", 0)
  t.shouldSend("session-2", 0)
  t.clearSession("session-1")
  expect(t.shouldSend("session-1", 1000)).toBe(true)
  expect(t.shouldSend("session-2", 1000)).toBe(false)
})

test("Throttle: interval of 0 means no throttling", () => {
  const t = new Throttle(0)
  expect(t.shouldSend("session-1", 0)).toBe(true)
  expect(t.shouldSend("session-1", 0)).toBe(true)
  expect(t.shouldSend("session-1", 1)).toBe(true)
})

test("Throttle: exact boundary case", () => {
  const t = new Throttle(100)
  t.shouldSend("session-1", 0)
  expect(t.shouldSend("session-1", 99)).toBe(false) // just before
  expect(t.shouldSend("session-1", 100)).toBe(true) // exactly at interval
})

test("Throttle: large time values", () => {
  const t = new Throttle(1000)
  t.shouldSend("session-1", 0)
  expect(t.shouldSend("session-1", 999)).toBe(false)
  expect(t.shouldSend("session-1", 1000)).toBe(true) // exactly at interval passes
  t.shouldSend("session-1", 1000) // call at exactly 1000
  expect(t.shouldSend("session-1", 2000)).toBe(true) // 2000 - 1000 = 1000, which is not < 1000
})
