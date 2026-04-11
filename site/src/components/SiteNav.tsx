'use client';

import { useEffect, useRef, useState } from 'react';
import { usePathname } from 'next/navigation';
import Link from 'next/link';

type NavLink = { readonly href: string; readonly label: string; readonly external?: boolean };

interface SiteNavProps {
  links: readonly NavLink[];
}

export function SiteNav({ links }: SiteNavProps) {
  const pathname = usePathname();
  const [open, setOpen] = useState(false);
  const closeButtonRef = useRef<HTMLButtonElement>(null);

  // ESC key closes drawer
  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false);
    }
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [open]);

  // Move focus into drawer when it opens
  useEffect(() => {
    if (open) closeButtonRef.current?.focus();
  }, [open]);

  // Lock body scroll while drawer is open
  useEffect(() => {
    document.body.style.overflow = open ? 'hidden' : '';
    return () => { document.body.style.overflow = ''; };
  }, [open]);

  function isActive(href: string): boolean {
    return pathname === href;
  }

  return (
    <>
      {/* ── Desktop nav (hidden on mobile via CSS) ── */}
      <nav aria-label="Primary" className="navDesktop">
        <ul className="navList">
          {links.map((link) => (
            <li key={link.href}>
              {link.external ? (
                <a href={link.href} rel="noreferrer" target="_blank">
                  {link.label}
                  <span className="srOnly"> (opens in new tab)</span>
                </a>
              ) : (
                <Link
                  href={link.href}
                  aria-current={isActive(link.href) ? 'page' : undefined}
                >
                  {link.label}
                </Link>
              )}
            </li>
          ))}
        </ul>
      </nav>

      {/* ── Hamburger button (hidden on desktop via CSS) ── */}
      <button
        className="navBurger"
        aria-label={open ? 'Close navigation menu' : 'Open navigation menu'}
        aria-expanded={open}
        aria-controls="site-nav-drawer"
        onClick={() => setOpen((v) => !v)}
      >
        {open ? <IconClose /> : <IconBurger />}
      </button>

      {/* ── Mobile drawer ── */}
      {open && (
        <>
          {/* Backdrop */}
          <div
            className="navOverlay"
            aria-hidden="true"
            onClick={() => setOpen(false)}
          />

          {/* Drawer panel */}
          <div
            id="site-nav-drawer"
            className="navDrawer"
            role="dialog"
            aria-modal="true"
            aria-label="Navigation menu"
          >
            <div className="navDrawerHeader">
              <span className="navDrawerWordmark" aria-hidden="true">
                <span className="wordmarkDot" />
                MEL
              </span>
              <button
                ref={closeButtonRef}
                className="navDrawerClose"
                aria-label="Close navigation menu"
                onClick={() => setOpen(false)}
              >
                <IconClose />
              </button>
            </div>

            <ul className="navDrawerLinks" role="list">
              {links.map((link) => (
                <li key={link.href}>
                  {link.external ? (
                    <a
                      href={link.href}
                      rel="noreferrer"
                      target="_blank"
                      onClick={() => setOpen(false)}
                    >
                      {link.label}
                      <span className="srOnly"> (opens in new tab)</span>
                    </a>
                  ) : (
                    <Link
                      href={link.href}
                      aria-current={isActive(link.href) ? 'page' : undefined}
                      onClick={() => setOpen(false)}
                    >
                      {link.label}
                    </Link>
                  )}
                </li>
              ))}
            </ul>
          </div>
        </>
      )}
    </>
  );
}

/* ── Icon primitives ── */

function IconBurger() {
  return (
    <svg
      width="15"
      height="12"
      viewBox="0 0 15 12"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      aria-hidden="true"
    >
      <line x1="0" y1="1"  x2="15" y2="1"  />
      <line x1="0" y1="6"  x2="15" y2="6"  />
      <line x1="0" y1="11" x2="15" y2="11" />
    </svg>
  );
}

function IconClose() {
  return (
    <svg
      width="14"
      height="14"
      viewBox="0 0 14 14"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      aria-hidden="true"
    >
      <line x1="1" y1="1"  x2="13" y2="13" />
      <line x1="13" y1="1" x2="1"  y2="13" />
    </svg>
  );
}
