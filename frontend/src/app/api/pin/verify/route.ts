import { NextRequest, NextResponse } from 'next/server';
import { verifyPinHash } from '@/lib/pinAuth';

export async function POST(req: NextRequest) {
  try {
    const { pin } = await req.json();
    if (!pin || !/^\d{4}$/.test(pin)) {
      return NextResponse.json({ ok: false }, { status: 400 });
    }

    const ok = verifyPinHash(pin);
    return NextResponse.json({ ok });
  } catch {
    return NextResponse.json({ ok: false }, { status: 500 });
  }
}
