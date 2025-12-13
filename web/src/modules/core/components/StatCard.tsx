import { Card, CardContent } from '@/components/ui/card';
import { Component, JSX } from 'solid-js';

interface StatCardProps {
  icon: JSX.Element;
  title: string;
  value: string | number;
  description?: string;
  color?: 'blue' | 'green' | 'orange' | 'red';
}

const StatCard: Component<StatCardProps> = (props) => {
  const colorClasses = () => {
    switch (props.color) {
      case 'blue':
        return 'bg-blue-100 text-blue-600';
      case 'green':
        return 'bg-green-100 text-green-600';
      case 'orange':
        return 'bg-orange-100 text-orange-600';
      case 'red':
        return 'bg-red-100 text-red-600';
      default:
        return 'bg-gray-100 text-gray-600';
    }
  };

  return (
    <Card>
      <CardContent class="p-6">
        <div class="flex items-center justify-between">
          <div class="flex-1">
            <p class="text-sm font-medium text-muted-foreground">{props.title}</p>
            <p class="mt-1 text-2xl font-bold text-card-foreground">{props.value}</p>
            {props.description && (
              <p class="mt-1 text-xs text-muted-foreground">{props.description}</p>
            )}
          </div>
          <div
            class={`flex size-12 items-center justify-center self-start rounded-lg ${colorClasses()}`}
          >
            {props.icon}
          </div>
        </div>
      </CardContent>
    </Card>
  );
};

export default StatCard;
