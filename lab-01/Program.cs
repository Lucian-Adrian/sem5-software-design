using System;
using System.Collections.Generic;
using System.Linq;
using TMPS.Behavioral;
using TMPS.Creational;
using TMPS.Domain;

namespace TMPS.Client;

internal static class Program
{
    private static int Main(string[] args)
    {
        if (args.Length == 0 || args[0] == "lab-01")
        {
            RunLab01();
            return 0;
        }

        if (args.Length == 1 && args[0] == "lab-03")
        {
            RunLab03();
            return 0;
        }

        Console.Error.WriteLine("Usage: TMPSLab01 [lab-01|lab-03]");
        return 1;
    }

    private static void RunLab01()
    {
        List<TaskItem> allTasks = new()
        {
            new TaskItem(1, "Write report", 3, new DateOnly(2026, 10, 12), false),
            new TaskItem(2, "Review notes", 2, new DateOnly(2026, 10, 11), false),
            new TaskItem(3, "Archive old draft", 1, new DateOnly(2026, 10, 10), true)
        };

        TaskReportOptions options = new TaskReportOptionsBuilder()
            .WithTitle("Study tasks")
            .WithMinimumPriority(2)
            .Build();

        List<TaskItem> selectedTasks = SelectTasks(allTasks, options);

        TaskReportCreator reportCreator = new CsvTaskReportCreator();
        Console.WriteLine(reportCreator.Build(options.Title, selectedTasks));

        TaskBoardRenderer board = new(new MarkdownBoardThemeFactory());
        Console.WriteLine(board.Render(options.Title, selectedTasks));
    }

    private static void RunLab03()
    {
        TaskBoard board = new(new List<TaskItem>
        {
            new(1, "Write report", 3, new DateOnly(2026, 10, 12), false),
            new(2, "Review notes", 2, new DateOnly(2026, 10, 11), false),
            new(3, "Archive old draft", 1, new DateOnly(2026, 10, 13), false)
        });

        TaskReportOptions options = new TaskReportOptionsBuilder()
            .WithTitle("Study tasks")
            .WithMinimumPriority(1)
            .Build();
        ITaskSortStrategy sortStrategy = new PrioritySortStrategy();

        board.Subscribe(new ConsoleTaskObserver());

        List<TaskItem> tasksBeforeCommand = SelectTasks(board.GetTasks(sortStrategy), options);
        TaskReportCreator reportCreator = new CsvTaskReportCreator();
        Console.WriteLine(reportCreator.Build(options.Title, tasksBeforeCommand));

        board.Execute(new CompleteTaskCommand(2));

        List<TaskItem> openTasks = SelectTasks(board.GetTasks(sortStrategy), options);
        TaskBoardRenderer boardRenderer = new(new MarkdownBoardThemeFactory());
        Console.WriteLine(boardRenderer.Render(options.Title, openTasks));
    }

    private static List<TaskItem> SelectTasks(
        IEnumerable<TaskItem> tasks,
        TaskReportOptions options)
    {
        return tasks
            .Where(task => options.IncludeCompleted || !task.IsCompleted)
            .Where(task => task.Priority >= options.MinimumPriority)
            .ToList();
    }
}
