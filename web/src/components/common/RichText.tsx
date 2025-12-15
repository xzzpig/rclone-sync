import { JSX, For, createMemo } from 'solid-js';

interface RichTextProps {
  text: string;
}

type TextNode =
  | { type: 'text'; content: string }
  | { type: 'strong'; content: string }
  | { type: 'a'; href: string; content: string };

/**
 * Parse text with simple HTML tags into an array of nodes
 * Supports: <strong>text</strong>, <a href="url">text</a>
 */
function parseText(text: string): TextNode[] {
  const nodes: TextNode[] = [];
  let remaining = text;
  let index = 0;

  while (index < remaining.length) {
    // Find next tag
    const tagMatch = remaining.slice(index).match(/<(strong|a[^>]*)>([^<]*)<\/(strong|a)>/);

    if (!tagMatch) {
      // No more tags, add remaining text
      if (index < remaining.length) {
        nodes.push({ type: 'text', content: remaining.slice(index) });
      }
      break;
    }

    // Add text before tag
    if (tagMatch.index! > 0) {
      nodes.push({ type: 'text', content: remaining.slice(index, index + tagMatch.index!) });
    }

    const fullTag = tagMatch[1];
    const content = tagMatch[2];

    if (fullTag === 'strong') {
      nodes.push({ type: 'strong', content });
    } else if (fullTag.startsWith('a')) {
      // Extract href from tag
      const hrefMatch = fullTag.match(/href="([^"]*)"/);
      const href = hrefMatch ? hrefMatch[1] : '';
      nodes.push({ type: 'a', href, content });
    }

    // Move index past this tag
    index += tagMatch.index! + tagMatch[0].length;
  }

  return nodes;
}

/**
 * RichText component - renders text with simple HTML tags as JSX
 *
 * Supports:
 * - <strong>text</strong> - renders as bold with primary color
 * - <a href="url">text</a> - renders as underlined link opening in new tab
 *
 * @example
 * <RichText text="Step 2: Configure your <strong>AWS S3</strong> connection." />
 * <RichText text="Use <a href=\"https://crontab.guru\">crontab.guru</a> for help." />
 */
export function RichText(props: RichTextProps): JSX.Element {
  const nodes = createMemo(() => parseText(props.text));

  return (
    <>
      <For each={nodes()}>
        {(node) => {
          switch (node.type) {
            case 'strong':
              return <strong class="font-bold text-primary">{node.content}</strong>;
            case 'a':
              return (
                <a href={node.href} target="_blank" rel="noopener noreferrer" class="underline">
                  {node.content}
                </a>
              );
            default:
              return <>{node.content}</>;
          }
        }}
      </For>
    </>
  );
}
