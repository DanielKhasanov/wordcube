import React from 'react';
import { Card, CardContent } from '@/components/ui/card';
import { GameCell } from './GameCell';
import { GameState, CellPosition, Direction } from '../types/game';

interface GameGridProps {
  gameState: GameState;
  selectedCell: CellPosition | null;
  selectedDirection: Direction;
  onCellClick: (position: CellPosition) => void;
  onKeyDown: (position: CellPosition, event: React.KeyboardEvent) => void;
}

export const GameGrid: React.FC<GameGridProps> = ({
  gameState,
  selectedCell,
  selectedDirection,
  onCellClick,
  onKeyDown,
}) => {
  const isHighlighted = (row: number, col: number): boolean => {
    if (!selectedCell) return false;
    if (selectedDirection === 'row') {
      return row === selectedCell.row;
    } else {
      return col === selectedCell.col;
    }
  };

  const isSelected = (row: number, col: number): boolean => {
    return selectedCell?.row === row && selectedCell?.col === col;
  };

  return (
    <Card className="bg-white shadow-lg">
      <CardContent className="p-6">
        <div className="grid grid-cols-5 gap-2">
          {gameState.grid.map((row, rowIndex) =>
            row.map((cell, colIndex) => (
              <GameCell
                key={`${rowIndex}-${colIndex}`}
                position={{ row: rowIndex, col: colIndex }}
                value={cell}
                isSelected={isSelected(rowIndex, colIndex)}
                isHighlighted={isHighlighted(rowIndex, colIndex)}
                selectedDirection={selectedDirection}
                onClick={onCellClick}
                onKeyDown={onKeyDown}
              />
            ))
          )}
        </div>
      </CardContent>
    </Card>
  );
};