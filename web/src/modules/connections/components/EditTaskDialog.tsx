import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { showToast } from '@/components/ui/toast';
import { Task } from '@/lib/types';
import { createEffect, createSignal } from 'solid-js';
import { TaskSettingsForm, TaskSettingsFormData } from './TaskSettingsForm';

interface EditTaskDialogProps {
  task: Task | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSave: (id: string, updates: Partial<Task>) => Promise<void>;
}

export function EditTaskDialog(props: EditTaskDialogProps) {
  const [formData, setFormData] = createSignal<TaskSettingsFormData>({
    name: '',
    direction: 'upload',
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
        name: task.name || '',
        direction: task.direction || 'upload',
        schedule: task.schedule ?? '',
        realtime: task.realtime || false,
        options: task.options || {},
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
        title: 'Task updated',
        description: `Task "${data.name}" has been updated successfully.`,
      });
      props.onOpenChange(false);
    } catch (error) {
      showToast({
        title: 'Failed to update task',
        description: error instanceof Error ? error.message : 'An unknown error occurred.',
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
          <DialogTitle>Edit Sync Task</DialogTitle>
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
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={isSubmitting()}>
            {isSubmitting() ? 'Saving...' : 'Save Changes'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
