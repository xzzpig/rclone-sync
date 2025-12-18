import { Button } from '@/components/ui/button';
import { TextField, TextFieldTextArea } from '@/components/ui/text-field';
import * as m from '@/paraglide/messages';
import { Component, createSignal } from 'solid-js';
import IconUpload from '~icons/lucide/upload';
import IconArrowRight from '~icons/lucide/arrow-right';

interface Step1InputProps {
  onNext: (content: string) => void;
  onCancel: () => void;
  loading?: boolean;
}

export const Step1Input: Component<Step1InputProps> = (props) => {
  const [content, setContent] = createSignal('');

  const handleFileUpload = async (e: Event) => {
    const target = e.target as HTMLInputElement;
    const file = target.files?.[0];
    if (file) {
      const text = await file.text();
      setContent(text);
    }
  };

  const handleNext = () => {
    if (content().trim()) {
      props.onNext(content());
    }
  };

  return (
    <div class="flex flex-col gap-4">
      <div class="text-sm text-muted-foreground">{m.import_step1Hint()}</div>

      <div class="relative">
        <TextField value={content()} onChange={setContent}>
          <TextFieldTextArea
            placeholder={`[remote-name]
type = drive
client_id = xxx
client_secret = xxx
token = {...}`}
            class="min-h-[200px] font-mono text-sm"
          />
        </TextField>
      </div>

      <div class="flex items-center gap-2">
        <Button variant="outline" size="sm" class="gap-2" as="label">
          <IconUpload class="size-4" />
          {m.import_uploadFile()}
          <input type="file" accept=".conf,.txt" class="hidden" onChange={handleFileUpload} />
        </Button>
        <span class="text-xs text-muted-foreground">{m.import_supportedFiles()}</span>
      </div>

      <div class="flex justify-end gap-2 pt-4">
        <Button variant="outline" onClick={props.onCancel}>
          {m.common_cancel()}
        </Button>
        <Button onClick={handleNext} disabled={!content().trim() || props.loading} class="gap-2">
          {props.loading ? m.import_parsing() : m.common_next()}
          <IconArrowRight class="size-4" />
        </Button>
      </div>
    </div>
  );
};
