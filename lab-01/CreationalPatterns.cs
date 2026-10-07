using System;
using System.Collections.Generic;
using TMPS.Domain;

namespace TMPS.Creational;

internal sealed record TaskReportOptions(
    string Title,
    bool IncludeCompleted,
    int MinimumPriority);

// Builder: set optional report settings one step at a time.
internal sealed class TaskReportOptionsBuilder
{
    private string _title = "Task report";
    private bool _includeCompleted;
    private int _minimumPriority = 1;

    public TaskReportOptionsBuilder WithTitle(string title)
    {
        _title = title;
        return this;
    }

    public TaskReportOptionsBuilder IncludeCompletedTasks()
    {
        _includeCompleted = true;
        return this;
    }

    public TaskReportOptionsBuilder WithMinimumPriority(int priority)
    {
        _minimumPriority = priority;
        return this;
    }

    public TaskReportOptions Build()
    {
        return new TaskReportOptions(_title, _includeCompleted, _minimumPriority);
    }
}

internal interface ITaskReportFormatter
{
    string Format(string title, IEnumerable<TaskItem> tasks);
}

// Factory Method: subclasses choose which formatter to create.
internal abstract class TaskReportCreator
{
    public string Build(string title, IEnumerable<TaskItem> tasks)
    {
        ITaskReportFormatter formatter = CreateFormatter();
        return formatter.Format(title, tasks);
    }

    protected abstract ITaskReportFormatter CreateFormatter();
}

internal sealed class CsvTaskReportCreator : TaskReportCreator
{
    protected override ITaskReportFormatter CreateFormatter()
    {
        return new CsvTaskReportFormatter();
    }
}

internal sealed class TextTaskReportCreator : TaskReportCreator
{
    protected override ITaskReportFormatter CreateFormatter()
    {
        return new TextTaskReportFormatter();
    }
}

internal sealed class CsvTaskReportFormatter : ITaskReportFormatter
{
    public string Format(string title, IEnumerable<TaskItem> tasks)
    {
        List<string> lines = new() { title, "Id,Title,Priority,Status" };

        foreach (TaskItem task in tasks)
        {
            string status = task.IsCompleted ? "Done" : "Open";
            lines.Add($"{task.Id},{Escape(task.Title)},{task.Priority},{status}");
        }

        return string.Join(Environment.NewLine, lines);
    }

    private static string Escape(string value)
    {
        return "\"" + value.Replace("\"", "\"\"") + "\"";
    }
}

internal sealed class TextTaskReportFormatter : ITaskReportFormatter
{
    public string Format(string title, IEnumerable<TaskItem> tasks)
    {
        List<string> lines = new() { title };

        foreach (TaskItem task in tasks)
        {
            string status = task.IsCompleted ? "done" : "open";
            lines.Add($"{task.Id}. {task.Title} (priority {task.Priority}, {status})");
        }

        return string.Join(Environment.NewLine, lines);
    }
}

internal interface IBoardHeader
{
    string Format(string title);
}

internal interface IBoardTaskRow
{
    string Format(TaskItem task);
}

internal interface IBoardThemeFactory
{
    IBoardHeader CreateHeader();
    IBoardTaskRow CreateTaskRow();
}

// Abstract Factory: create a matching family of header and row formatters.
internal sealed class PlainTextBoardThemeFactory : IBoardThemeFactory
{
    public IBoardHeader CreateHeader() => new PlainTextBoardHeader();
    public IBoardTaskRow CreateTaskRow() => new PlainTextBoardTaskRow();
}

internal sealed class MarkdownBoardThemeFactory : IBoardThemeFactory
{
    public IBoardHeader CreateHeader() => new MarkdownBoardHeader();
    public IBoardTaskRow CreateTaskRow() => new MarkdownBoardTaskRow();
}

internal sealed class PlainTextBoardHeader : IBoardHeader
{
    public string Format(string title) => title;
}

internal sealed class PlainTextBoardTaskRow : IBoardTaskRow
{
    public string Format(TaskItem task)
    {
        string status = task.IsCompleted ? "done" : "open";
        return $"{task.Id}. {task.Title} (priority {task.Priority}, {status})";
    }
}

internal sealed class MarkdownBoardHeader : IBoardHeader
{
    public string Format(string title) => $"# {title}";
}

internal sealed class MarkdownBoardTaskRow : IBoardTaskRow
{
    public string Format(TaskItem task)
    {
        string marker = task.IsCompleted ? "x" : " ";
        return $"- [{marker}] {task.Title} (P{task.Priority})";
    }
}

internal sealed class TaskBoardRenderer
{
    private readonly IBoardHeader _header;
    private readonly IBoardTaskRow _taskRow;

    public TaskBoardRenderer(IBoardThemeFactory factory)
    {
        _header = factory.CreateHeader();
        _taskRow = factory.CreateTaskRow();
    }

    public string Render(string title, IEnumerable<TaskItem> tasks)
    {
        List<string> lines = new() { _header.Format(title) };

        foreach (TaskItem task in tasks)
        {
            lines.Add(_taskRow.Format(task));
        }

        return string.Join(Environment.NewLine, lines);
    }
}
