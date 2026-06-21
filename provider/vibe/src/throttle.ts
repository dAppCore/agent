// SPDX-License-Identifier: EUPL-1.2

// Throttle gates per-session progress reports to at most one per interval.
// Time is passed in (not read from a clock) so the gate is deterministic and
// unit-testable without faking timers.
export class Throttle {
  private readonly last = new Map<string, number>()

  constructor(private readonly intervalMs: number) {}

  // shouldSend reports whether a progress message for sessionId may be sent at
  // time `now` (ms). The first call for a session always passes; subsequent
  // calls within intervalMs of the last accepted send are blocked. Accepting a
  // send records `now` as the new baseline.
  //
  //   const t = new Throttle(60000)
  //   t.shouldSend("s", 0)      // true
  //   t.shouldSend("s", 30000)  // false
  //   t.shouldSend("s", 61000)  // true
  shouldSend(sessionId: string, now: number): boolean {
    const prev = this.last.get(sessionId)
    if (prev !== undefined && now - prev < this.intervalMs) {
      return false
    }
    this.last.set(sessionId, now)
    return true
  }

  // clear removes all recorded timestamps, resetting the throttle state.
  clear(): void {
    this.last.clear()
  }

  // clearSession removes the recorded timestamp for a specific session.
  clearSession(sessionId: string): void {
    this.last.delete(sessionId)
  }
}
