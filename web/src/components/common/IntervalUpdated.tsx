import { Component, createEffect, createSignal, JSX, mergeProps, onCleanup } from 'solid-js';

interface IntervalUpdatedProps {
  when?: boolean;
  interval?: number;
  children: (now: number) => JSX.Element;
}

const IntervalUpdated: Component<IntervalUpdatedProps> = (_props) => {
  const props = mergeProps({ enabled: true, interval: 1000 }, _props);
  const [now, setNow] = createSignal(Date.now());

  createEffect(() => {
    if (props.when) {
      // Update immediately on enable if needed, or just wait for interval.
      // To ensure freshness, we can update immediately.
      setNow(Date.now());

      const timer = setInterval(() => {
        setNow(Date.now());
      }, props.interval);
      onCleanup(() => clearInterval(timer));
    }
  });

  return <>{props.children(now())}</>;
};

export default IntervalUpdated;
