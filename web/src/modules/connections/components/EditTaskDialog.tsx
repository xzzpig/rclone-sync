import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { showToast } from '@/components/ui/toast';
import { type Task, type UpdateTaskInput } from '@/lib/types';
import * as m from '@/paraglide/messages.js';
import { createEffect, createSignal } from 'solid-js';
import { TaskSettingsForm, taskToUpdateInput } from './TaskSettingsForm';

interface EditTaskDialogProps {
  task: Task | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSave: (id: string, updates: UpdateTaskInput) => Promise<void>;
}

export function EditTaskDialog(props: EditTaskDialogProps) {
  const [formData, setFormData] = createSignal<UpdateTaskInput>({
    name: '',
    direction: 'UPLOAD',
    schedule: '',
    realtime: false,
    options: {},
  });
  const [isSubmitting, setIsSubmitting] = createSignal(false);

  // Populate form when task changes
  createEffect(() => {
    const task = props.task;
    if (task) {
      setFormData(taskToUpdateInput(task));
    }
  });

  const handleSave = async () => {
    const task = props.task;
    if (!task) return;

    setIsSubmitting(true);
    try {
      await props.onSave(task.id, formData());
      showToast({
        title: m.toast_taskUpdated(),
        description: m.toast_taskUpdatedDesc({ name: formData().name ?? '' }),
      });
      props.onOpenChange(false);
    } catch (error) {
      showToast({
        title: m.toast_failedToUpdateTask(),
        description: error instanceof Error ? error.message : m.error_unknownError(),
        variant: 'destructive',
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent class="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>{m.wizard_editTask()}</DialogTitle>
        </DialogHeader>
        <div class="py-4">
          <TaskSettingsForm
            value={formData()}
            onChange={setFormData}
            connectionId={props.task?.connection?.id}
            remotePath={props.task?.remotePath}
            sourcePath={props.task?.sourcePath}
          />
        </div>
        <DialogFooter>
          <Button
            variant="secondary"
            onClick={() => props.onOpenChange(false)}
            disabled={isSubmitting()}
          >
            {m.common_cancel()}
          </Button>
          <Button onClick={handleSave} disabled={isSubmitting()}>
            {isSubmitting() ? m.wizard_saving() : m.wizard_saveChanges()}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
