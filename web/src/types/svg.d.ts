declare module '*.svg' {
  import type { Component, JSX } from 'solid-js';
  const component: Component<JSX.SvgSVGAttributes<SVGSVGElement>>;
  export default component;
}

declare module '*.svg?component' {
  import type { Component, JSX } from 'solid-js';
  const component: Component<JSX.SvgSVGAttributes<SVGSVGElement>>;
  export default component;
}

declare module '*.svg?url' {
  const url: string;
  export default url;
}

declare module '*.svg?raw' {
  const raw: string;
  export default raw;
}
