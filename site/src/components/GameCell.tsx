import React, { useRef, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { CellPosition, Direction } from '../types/game';
import { cn } from '@/lib/utils';

interface GameCellProps {
  position: CellPosition;
  value: string;
  isSelected: boolean;
  isHighlighted: boolean;
  selectedDirection: Direction;
  onClick: (position: CellPosition) => void;
  onKeyDown: (position: CellPosition, event: React.KeyboardEvent) => void;
}

export const GameCell: React.FC<GameCellProps> = ({
  position,
  value,
  isSelected,
  isHighlighted,
  onClick,
  onKeyDown,
}) => {
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (isSelected && inputRef.current) {
      inputRef.current.focus();
      // Set cursor to the beginning
      inputRef.current.setSelectionRange(0, 0);
    }
  }, [isSelected]);

  const handleClick = () => {
    onClick(position);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    onKeyDown(position, e);
  };

  const handleChange = () => {
    // No-op: We handle all input via onKeyDown
    // This prevents the React warning about missing onChange
  };

  return (
    <Button
      variant="outline"
      size="icon"
      className={cn(
        'aspect-square hover:bg-green-200',
        isHighlighted && 'bg-blue-200' //hover:bg-blue-200',
      )}
      onClick={handleClick}
      asChild
    >
      <Input
        ref={inputRef}
        type="text"
        value={value}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        className="w-auto h-auto max-w-36 max-h-36 text-center text-3xl font-bold"
        maxLength={1}
        aria-label={`Cell ${position.row + 1}, ${position.col + 1}`}
      />
    </Button>
  );
};