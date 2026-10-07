using System;
using System.Collections.Generic;
using System.Linq;
using TMPS.Domain;

namespace TMPS.Behavioral;

internal interface ITaskCommand
{
    void Execute(TaskBoard board);
}

internal sealed class CompleteTaskCommand : ITaskCommand
{
    private readonly int _taskId;

    public CompleteTaskCommand(int taskId)
    {
        _taskId = taskId;
    }

    public void Execute(TaskBoard board)
    {
        board.CompleteTask(_taskId);
    }
}

internal interface ITaskObserver
{
    void OnTaskCompleted(TaskItem task);
}

internal sealed class ConsoleTaskObserver : ITaskObserver
{
    public void OnTaskCompleted(TaskItem task)
    {
        Console.WriteLine($"Observer: task {task.Id} was completed.");
    }
}

internal sealed class TaskBoard
{
    private readonly List<TaskItem> _tasks;
    private readonly List<ITaskObserver> _observers = new();

    public TaskBoard(IEnumerable<TaskItem> tasks)
    {
        _tasks = tasks.ToList();
    }

    public void Subscribe(ITaskObserver observer)
    {
        _observers.Add(observer);
    }

    public void Execute(ITaskCommand command)
    {
        command.Execute(this);
    }

    public void CompleteTask(int taskId)
    {
        foreach (TaskItem task in _tasks)
        {
            if (task.Id == taskId)
            {
                task.MarkComplete();
                NotifyObservers(task);
                return;
            }
        }

        throw new InvalidOperationException($"Task {taskId} was not found.");
    }

    public IEnumerable<TaskItem> GetTasks(ITaskSortStrategy strategy)
    {
        return strategy.Sort(_tasks);
    }

    private void NotifyObservers(TaskItem task)
    {
        foreach (ITaskObserver observer in _observers)
        {
            observer.OnTaskCompleted(task);
        }
    }
}

internal interface ITaskSortStrategy
{
    IEnumerable<TaskItem> Sort(IEnumerable<TaskItem> tasks);
}

internal sealed class PrioritySortStrategy : ITaskSortStrategy
{
    public IEnumerable<TaskItem> Sort(IEnumerable<TaskItem> tasks)
    {
        return tasks
            .OrderByDescending(task => task.Priority)
            .ThenBy(task => task.DueDate);
    }
}

internal sealed class DueDateSortStrategy : ITaskSortStrategy
{
    public IEnumerable<TaskItem> Sort(IEnumerable<TaskItem> tasks)
    {
        return tasks
            .OrderBy(task => task.DueDate)
            .ThenByDescending(task => task.Priority);
    }
}
