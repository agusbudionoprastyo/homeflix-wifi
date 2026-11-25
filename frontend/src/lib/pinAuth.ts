import { createHash, timingSafeEqual } from 'crypto';
import { readFileSync, writeFileSync } from 'fs';
import { homedir } from 'os';
import { join } from 'path';

const HASH_FILE = process.env.SETTINGS_PIN_HASH_FILE || join(homedir(), '.homeflix-pin-hash');

/** Returns the hex SHA-256 digest of a PIN string. */
export function sha256Pin(value: string): string {
  return createHash('sha256').update(value).digest('hex');
}

/** Loads the expected PIN hash from disk, falling back to SETTINGS_PIN. */
export function getExpectedPinHash(): string {
  try {
    return readFileSync(HASH_FILE, 'utf-8').trim();
  } catch {
    const pin = process.env.SETTINGS_PIN;
    if (!pin) throw new Error('SETTINGS_PIN not set');
    return sha256Pin(pin);
  }
}

/** Compares a candidate 4-digit PIN against the expected hash in constant time. */
export function verifyPinHash(pin: string): boolean {
  const entered = Buffer.from(sha256Pin(pin));
  const expected = Buffer.from(getExpectedPinHash());
  return entered.length === expected.length && timingSafeEqual(entered, expected);
}

/** Persists a new 4-digit PIN hash to disk with owner-only permissions. */
export function savePinHash(newPin: string): void {
  writeFileSync(HASH_FILE, sha256Pin(newPin), { mode: 0o600 });
}
