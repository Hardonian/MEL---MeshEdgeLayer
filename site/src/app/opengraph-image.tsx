import { ImageResponse } from 'next/og';

export const runtime = 'edge';

export const alt = 'MEL — MeshEdgeLayer';
export const size = { width: 1200, height: 630 };
export const contentType = 'image/png';

export default function OpenGraphImage() {
  return new ImageResponse(
    (
      <div
        style={{
          height: '100%',
          width: '100%',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'center',
          padding: 64,
          background: 'linear-gradient(165deg, #121821 0%, #0b0f13 100%)',
          border: '1px solid #223043',
        }}
      >
        <div
          style={{
            fontSize: 28,
            fontFamily: 'monospace',
            color: '#7aa2f7',
            letterSpacing: '0.12em',
            marginBottom: 24,
          }}
        >
          MESHEDGELAYER
        </div>
        <div
          style={{
            fontSize: 56,
            fontWeight: 700,
            color: '#d4dde7',
            lineHeight: 1.15,
            maxWidth: 900,
          }}
        >
          Incident intelligence bounded by runtime evidence.
        </div>
        <div
          style={{
            marginTop: 32,
            fontSize: 24,
            color: '#94a1b3',
            maxWidth: 800,
          }}
        >
          Evidence-first · local-first · explicit degraded states
        </div>
      </div>
    ),
    { ...size }
  );
}
