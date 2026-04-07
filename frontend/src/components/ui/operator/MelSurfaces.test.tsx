import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import {
  MelDenseRow,
  MelPanelInset,
  MelPanelSection,
  MelPageSection,
  MelSegment,
  MelSegmentItem,
  MelTruthBadge,
} from './MelSurfaces'

describe('MelSurfaces', () => {
  it('MelPanelInset applies tone class for dense list chrome', () => {
    render(
      <MelPanelInset tone="dense" data-testid="dense-inset">
        row
      </MelPanelInset>,
    )
    const el = screen.getByTestId('dense-inset')
    expect(el.className).toContain('mel-panel-inset')
    expect(el.className).toContain('border-border/50')
  })

  it('MelPanelInset applies tone class for degraded', () => {
    const { container } = render(
      <MelPanelInset tone="degraded" data-testid="inset">
        content
      </MelPanelInset>,
    )
    const el = screen.getByTestId('inset')
    expect(el.className).toContain('mel-panel-inset')
    expect(el.className).toContain('border-signal-degraded/30')
    expect(container.textContent).toContain('content')
  })

  it('MelPageSection wires aria-labelledby when id and title present', () => {
    render(
      <MelPageSection id="sec-a" title="Section title">
        <p>body</p>
      </MelPageSection>,
    )
    const section = document.getElementById('sec-a')
    expect(section?.getAttribute('aria-labelledby')).toBe('sec-a-heading')
    expect(document.getElementById('sec-a-heading')?.textContent).toBe('Section title')
  })

  it('MelSegment radiogroup wires group and segment clicks', () => {
    const onA = vi.fn()
    const onB = vi.fn()
    render(
      <MelSegment label="Pick" radiogroupLabel="Choose option">
        <MelSegmentItem active onClick={onA}>
          A
        </MelSegmentItem>
        <MelSegmentItem onClick={onB}>B</MelSegmentItem>
      </MelSegment>,
    )
    const group = screen.getByRole('radiogroup', { name: 'Choose option' })
    expect(group).toBeTruthy()
    const buttons = screen.getAllByRole('button')
    expect(buttons).toHaveLength(2)
    fireEvent.click(buttons[1]!)
    expect(onB).toHaveBeenCalled()
  })

  it('MelTruthBadge maps complete semantic to complete variant styling', () => {
    const { container } = render(<MelTruthBadge semantic="complete">OK</MelTruthBadge>)
    const badge = container.querySelector('.border-signal-complete\\/35')
    expect(badge).toBeTruthy()
  })

  it('MelPanelSection renders chrome title and body region', () => {
    render(
      <MelPanelSection heading="Panel title" description="Helper" data-testid="mps">
        <p>inner</p>
      </MelPanelSection>,
    )
    expect(screen.getByText('Panel title')).toBeTruthy()
    expect(screen.getByText('Helper')).toBeTruthy()
    expect(screen.getByText('inner')).toBeTruthy()
  })

  it('MelDenseRow applies warning tone classes', () => {
    render(
      <MelDenseRow tone="warning" data-testid="row">
        x
      </MelDenseRow>,
    )
    const el = screen.getByTestId('row')
    expect(el.className).toContain('border-signal-partial')
  })
})
