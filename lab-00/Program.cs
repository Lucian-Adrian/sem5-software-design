using System;
using System.Collections.Generic;
using System.Linq;

internal static class Program
{
    private static void Main()
    {
        List<TaskItem> tasks = new()
        {
            new TaskItem(1, "Read the brief", 2, false),
            new TaskItem(2, "Submit the draft", 3, true),
            new TaskItem(3, "Prepare questions", 1, false)
        };

        ITaskSource source = new InMemoryTaskSource(tasks);
        ITaskFilter filter = new OpenTaskFilter();
        TaskReport report = new(source, filter);
        report.Print();
    }
}

internal sealed record TaskItem(int Id, string Title, int Priority, bool IsDone);

internal interface ITaskSource
{
    IReadOnlyList<TaskItem> GetAll();
}

internal sealed class InMemoryTaskSource : ITaskSource
{
    private readonly IReadOnlyList<TaskItem> _tasks;

    public InMemoryTaskSource(IReadOnlyList<TaskItem> tasks)
    {
        _tasks = tasks;
    }

    public IReadOnlyList<TaskItem> GetAll()
    {
        return _tasks;
    }
}

internal interface ITaskFilter
{
    IEnumerable<TaskItem> Apply(IEnumerable<TaskItem> tasks);
}

internal sealed class OpenTaskFilter : ITaskFilter
{
    public IEnumerable<TaskItem> Apply(IEnumerable<TaskItem> tasks)
    {
        return tasks.Where(task => !task.IsDone);
    }
}

internal sealed class TaskReport
{
    private readonly ITaskSource _source;
    private readonly ITaskFilter _filter;

    public TaskReport(ITaskSource source, ITaskFilter filter)
    {
        _source = source;
        _filter = filter;
    }

    public void Print()
    {
        List<TaskItem> openTasks = _filter.Apply(_source.GetAll()).ToList();
        Console.WriteLine($"Open tasks: {openTasks.Count}");

        foreach (TaskItem task in openTasks)
        {
            Console.WriteLine($"{task.Id}: {task.Title} (priority {task.Priority})");
        }
    }
}
