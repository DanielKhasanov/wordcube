import React from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { ScrollArea } from '@/components/ui/scroll-area';
import { WordSquare } from '@/components/ui/word-square';
import { useSolutions } from '../hooks/useSolutions';
import { GameState } from '../types/game';

interface SolutionsSidebarProps {
  gameState: GameState;
}

export const SolutionsSidebar: React.FC<SolutionsSidebarProps> = ({ gameState }) => {
  console.log('SolutionsSidebar - gameState.grid:', gameState.grid);
  const { data: solutions, isLoading, error } = useSolutions(gameState.grid);

  const getTitle = () => {
    const count = solutions?.length || 0;
    const loading = isLoading ? (count === 0 ? '...' : ' - Loading more...') : '';
    return `Word Solutions (${count})${loading}`;
  };

  const renderContent = () => {
    if (error) {
      return (
        <div className="flex items-center justify-center py-8">
          <div className="text-sm text-red-600">Failed to load solutions: {error.message}</div>
        </div>
      );
    }

    if (isLoading && !solutions?.length) {
      return (
        <div className="flex items-center justify-center py-8">
          <div className="text-sm text-muted-foreground">Loading solutions...</div>
        </div>
      );
    }

    return (
      <div className="space-y-4">
        {solutions?.map((solution, index) => (
          <div key={`solution-${index}`} className="space-y-2">
            <h3 className="text-sm font-medium text-muted-foreground">
              Solution {solution.id}
            </h3>
            <WordSquare
              initialGrid={solution.grid}
              size="sm"
              variant="default"
              className="mx-auto"
            />
          </div>
        ))}
        {isLoading && solutions?.length && (
          <div className="flex items-center justify-center py-4">
            <div className="text-xs text-muted-foreground">Loading more solutions...</div>
          </div>
        )}
      </div>
    );
  };

  return (
    <Card className="bg-white shadow-lg">
      <CardHeader className="pb-3">
        <CardTitle className="text-lg">{getTitle()}</CardTitle>
      </CardHeader>
      <CardContent>
        <ScrollArea className="h-[calc(100vh-200px)]">
          {renderContent()}
        </ScrollArea>
      </CardContent>
    </Card>
  );
};