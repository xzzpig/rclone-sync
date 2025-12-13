import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { JSX, ParentComponent } from 'solid-js';

interface ConnectionViewLayoutProps {
  title: JSX.Element;
  actions?: JSX.Element;
}

const ConnectionViewLayout: ParentComponent<ConnectionViewLayoutProps> = (props) => {
  return (
    <div class="flex h-full flex-col">
      <Card class="flex min-h-0 flex-1 flex-col">
        <CardHeader class="shrink-0">
          <div class="flex items-center justify-between">
            <CardTitle>{props.title}</CardTitle>
            <div class="flex flex-wrap items-center gap-2">{props.actions}</div>
          </div>
        </CardHeader>
        <CardContent class="flex min-h-0 min-w-0 flex-1 flex-col">{props.children}</CardContent>
      </Card>
    </div>
  );
};

export default ConnectionViewLayout;
