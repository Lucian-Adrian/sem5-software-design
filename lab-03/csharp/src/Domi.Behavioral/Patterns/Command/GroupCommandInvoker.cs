using Domi.Behavioral.Patterns.Observer;

namespace Domi.Behavioral.Patterns.Command;

/// <summary>
/// INVOKER: queues commands, executes them through IGroupCommand, maintains
/// a LIFO undo history of successful commands, and notifies observers.
/// </summary>
public sealed class GroupCommandInvoker(IGroupActivitySubject notifier)
{
    private readonly Queue<IGroupCommand> _queue = new();
    private readonly Stack<IGroupCommand> _history = new();

    public int QueuedCount => _queue.Count;
    public int HistoryCount => _history.Count;

    public void Enqueue(IGroupCommand command)
    {
        ArgumentNullException.ThrowIfNull(command);
        _queue.Enqueue(command);
    }

    public GroupActivityEvent Run(IGroupCommand command)
    {
        ArgumentNullException.ThrowIfNull(command);
        var activity = command.Execute();
        _history.Push(command);
        notifier.Notify(activity);
        return activity;
    }

    public IReadOnlyList<GroupActivityEvent> RunQueued()
    {
        var events = new List<GroupActivityEvent>();
        while (_queue.Count > 0)
            events.Add(Run(_queue.Dequeue()));
        return events;
    }

    public GroupActivityEvent UndoLast()
    {
        if (_history.Count == 0)
            throw new InvalidOperationException("no executed commands to undo");

        var command = _history.Peek();
        var activity = command.Undo();
        _history.Pop();
        notifier.Notify(activity);
        return activity;
    }
}
