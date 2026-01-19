import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { type HookInput } from '@/lib/types';
import * as m from '@/paraglide/messages.js';
import { createEffect, createSignal } from 'solid-js';
import { HookForm } from './HookForm';

const defaultFormData: HookInput = {
  enabled: true,
  priority: null,
  event: 'ON_SUCCESS',
  type: 'HTTP',
  onError: 'IGNORE',
  config: {
    url: null,
    method: 'POST',
    headers: null,
    body: null,
    command: null,
    workDir: null,
    timeout: null,
  },
};

interface HookDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSave: (data: HookInput) => Promise<void> | void;
  initialData?: HookInput;
  isEdit?: boolean;
}

export function HookDialog(props: HookDialogProps) {
  const [formData, setFormData] = createSignal<HookInput>(defaultFormData);
  const [isSubmitting, setIsSubmitting] = createSignal(false);
  const [isValid, setIsValid] = createSignal(true);

  createEffect(() => {
    if (props.open) {
      if (props.initialData) {
        setFormData(props.initialData);
      } else {
        setFormData(defaultFormData);
      }
    }
  });

  const handleSubmit = async () => {
    if (!isValid()) return;

    setIsSubmitting(true);
    try {
      await props.onSave(formData());
      props.onOpenChange(false);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent class="max-h-[90vh] overflow-y-auto sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>{props.isEdit ? m.hook_editHook() : m.hook_createHook()}</DialogTitle>
        </DialogHeader>
        <div class="py-4">
          <HookForm value={formData()} onChange={setFormData} onValidate={setIsValid} />
        </div>
        <DialogFooter>
          <Button
            variant="secondary"
            onClick={() => props.onOpenChange(false)}
            disabled={isSubmitting()}
          >
            {m.common_cancel()}
          </Button>
          <Button onClick={handleSubmit} disabled={isSubmitting() || !isValid()}>
            {isSubmitting() ? m.common_loading() : m.common_save()}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
