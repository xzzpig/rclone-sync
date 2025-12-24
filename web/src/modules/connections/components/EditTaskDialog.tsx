import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { showToast } from '@/components/ui/toast';
import { type SyncDirection, type Task, type UpdateTaskInput } from '@/lib/types';
import * as m from '@/paraglide/messages.js';
import { createEffect, createSignal } from 'solid-js';
import { TaskSettingsForm, TaskSettingsFormData } from './TaskSettingsForm';

interface EditTaskDialogProps {
  task: Task | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSave: (id: string, updates: UpdateTaskInput) => Promise<void>;
}

export function EditTaskDialog(props: EditTaskDialogProps) {
  const [formData, setFormData] = createSignal<TaskSettingsFormData>({
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
      setFormData({
        name: task.name ?? '',
        direction: task.direction as SyncDirection,
        schedule: task.schedule ?? '',
        realtime: task.realtime ?? false,
        options: {},
      });
    }
  });

  const handleSave = async () => {
    const task = props.task;
    if (!task) return;

    setIsSubmitting(true);
    try {
      const data = formData();
      await props.onSave(task.id, {
        name: data.name,
        direction: data.direction,
        schedule: data.schedule,
        realtime: data.realtime,
        options: data.options,
      });
      showToast({
        title: m.toast_taskUpdated(),
        description: m.toast_taskUpdatedDesc({ name: data.name }),
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
          <TaskSettingsForm value={formData()} onChange={setFormData} />
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
