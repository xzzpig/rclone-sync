import { Component } from 'solid-js';
import { Skeleton } from '@/components/ui/skeleton';

export const CardSkeleton: Component = () => {
    return (
        <div class="flex flex-col space-y-3">
            <Skeleton class="h-[125px] w-[250px] rounded-xl" />
            <div class="space-y-2">
                <Skeleton class="h-4 w-[250px]" />
                <Skeleton class="h-4 w-[200px]" />
            </div>
        </div>
    );
};

export const ListSkeleton: Component<{ count?: number }> = (props) => {
    const count = props.count || 3;
    return (
        <div class="space-y-2">
            {Array.from({ length: count }).map(() => (
                <div class="flex items-center space-x-4">
                    <Skeleton class="h-12 w-12 rounded-full" />
                    <div class="space-y-2">
                        <Skeleton class="h-4 w-[250px]" />
                        <Skeleton class="h-4 w-[200px]" />
                    </div>
                </div>
            ))}
        </div>
    );
};
