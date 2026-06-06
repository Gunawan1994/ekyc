

interface CardProps {
  title?: string
  children: React.ReactNode
  className?: string
  /** Renders the card in a compact stats style with less padding */
  stats?: boolean
}

export function Card({ title, children, className = '', stats = false }: CardProps) {
  return (
    <div
      className={[
        'bg-white rounded-xl border border-slate-200 shadow-sm',
        stats ? 'p-5' : 'p-6',
        className,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      {title && (
        <h3
          className={[
            'font-semibold text-slate-800',
            stats ? 'text-sm mb-3' : 'text-base mb-4',
          ].join(' ')}
        >
          {title}
        </h3>
      )}
      {children}
    </div>
  )
}
