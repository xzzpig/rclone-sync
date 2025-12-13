import type { ValidComponent } from "solid-js"
import { splitProps } from "solid-js"

import type { PolymorphicProps } from "@kobalte/core/polymorphic"
import * as SkeletonPrimitive from "@kobalte/core/skeleton"

import { cn } from "@/lib/utils"

type SkeletonRootProps<T extends ValidComponent = "div"> =
  SkeletonPrimitive.SkeletonRootProps<T> & { class?: string | undefined }

const Skeleton = <T extends ValidComponent = "div">(
  props: PolymorphicProps<T, SkeletonRootProps<T>>
) => {
  const [local, others] = splitProps(props as SkeletonRootProps, [
    "class",
    "width",
    "height",
  ])
  return (
    <SkeletonPrimitive.Root
      width={local.width}
      height={local.height}
      class={cn("bg-primary/10 data-[animate='true']:animate-pulse", local.class)}
      {...others}
      style={{
        // 如果没有提供 width/height 属性，强制移除 Kobalte 默认的内联样式 (100%/auto)
        // 从而允许 Tailwind 类 (w-*, h-*) 生效
        // @ts-ignore
        width: local.width ? undefined : null,
        // @ts-ignore
        height: local.height ? undefined : null,
        // 保留用户传入的 style
        // @ts-ignore
        ...others.style,
      }}
    />
  )
}

export { Skeleton }
