import { NextRequest, NextResponse } from 'next/server';
import { savePinHash, verifyPinHash } from '@/lib/pinAuth';

export async function POST(req: NextRequest) {
  try {
    const { currentPin, newPin } = await req.json();
    if (!currentPin || !/^\d{4}$/.test(currentPin) || !newPin || !/^\d{4}$/.test(newPin)) {
      return NextResponse.json({ ok: false, error: 'Invalid PIN format' }, { status: 400 });
    }

    if (!verifyPinHash(currentPin)) {
      return NextResponse.json({ ok: false, error: 'Current PIN is incorrect' });
    }

    savePinHash(newPin);
    return NextResponse.json({ ok: true });
  } catch {
    return NextResponse.json({ ok: false, error: 'Server error' }, { status: 500 });
  }
}
