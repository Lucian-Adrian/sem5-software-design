using System;

namespace TMPS.Domain;

internal sealed class TaskItem
{
    public TaskItem(int id, string title, int priority, DateOnly dueDate, bool isCompleted)
    {
        Id = id;
        Title = title;
        Priority = priority;
        DueDate = dueDate;
        IsCompleted = isCompleted;
    }

    public int Id { get; }
    public string Title { get; }
    public int Priority { get; }
    public DateOnly DueDate { get; }
    public bool IsCompleted { get; private set; }

    public void MarkComplete()
    {
        IsCompleted = true;
    }
}
