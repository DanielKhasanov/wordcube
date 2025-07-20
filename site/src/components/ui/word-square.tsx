"use client"

import * as React from "react"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { RotateCw } from "lucide-react"
import { cva, type VariantProps } from "class-variance-authority"

const wordSquareVariants = cva(
  "inline-flex flex-col gap-0.5 p-2 rounded-md border bg-card",
  {
    variants: {
      variant: {
        default: "border-border",
        ghost: "border-transparent bg-transparent",
      },
      size: {
        sm: "p-1.5 gap-0.5",
        default: "p-2 gap-0.5",
        lg: "p-3 gap-1",
        xl: "p-3 gap-1",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

const cellVariants = cva(
  "flex items-center justify-center font-mono font-medium transition-colors duration-150 border border-border/50 bg-background",
  {
    variants: {
      size: {
        sm: "w-6 h-6 text-xs rounded-sm",
        default: "w-8 h-8 text-sm rounded",
        lg: "w-10 h-10 text-base rounded-md",
        xl: "w-16 h-16 text-base rounded-lg",
      },
      highlighted: {
        true: "bg-primary/10 border-primary/30 text-primary",
        false: "hover:bg-muted/50",
      },
    },
    defaultVariants: {
      size: "default",
      highlighted: false,
    },
  }
)

export interface WordSquareProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof wordSquareVariants> {
  initialGrid?: string[][]
  highlightedCells?: Array<{ row: number; col: number }>
  onTranspose?: (transposedGrid: string[][]) => void
}

const WordSquare = React.forwardRef<HTMLDivElement, WordSquareProps>(
  ({ className, variant, size, initialGrid, highlightedCells = [], onTranspose, ...props }, ref) => {
    const defaultGrid = Array(5).fill(null).map(() => Array(5).fill(''))
    const [currentGrid, setCurrentGrid] = React.useState(initialGrid || defaultGrid)

    React.useEffect(() => {
      if (initialGrid) {
        setCurrentGrid(initialGrid)
      }
    }, [initialGrid])

    const isCellHighlighted = (row: number, col: number) => {
      return highlightedCells.some(cell => cell.row === row && cell.col === col)
    }

    const handleTranspose = () => {
      const transposed = currentGrid[0].map((_, colIndex) =>
        currentGrid.map(row => row[colIndex])
      )
      setCurrentGrid(transposed)
      onTranspose?.(transposed)
    }

    return (
    <div className="relative inline-block">
        <div
          ref={ref}
          className={cn(wordSquareVariants({ variant, size, className }))}
          {...props}
        >
          {currentGrid.map((row, rowIndex) => (
            <div key={rowIndex} className="flex gap-0.5">
              {row.map((cell, colIndex) => (
                <div
                  key={`${rowIndex}-${colIndex}`}
                  className={cn(
                    cellVariants({
                      size,
                      highlighted: isCellHighlighted(rowIndex, colIndex),
                    })
                  )}
                >
                  {cell}
                </div>
              ))}
            </div>
          ))}
        </div>
        
        {/* Transpose Button */}
        <Button
          variant="ghost"
          size="sm"
          className="absolute -top-2 -right-2 h-6 w-6 p-0 rounded-full bg-muted border border-border shadow-sm hover:bg-muted"
          onClick={handleTranspose}
          title="Transpose word square"
        >
          <RotateCw className="h-3 w-3" />
          <span className="sr-only">Transpose</span>
        </Button>
      </div>
    )
  }
)

WordSquare.displayName = "WordSquare"

export { WordSquare, wordSquareVariants }