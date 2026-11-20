namespace Domi.Behavioral.Patterns.Observer;

/// <summary>
/// CONCRETE SUBJECT: holds only IGroupActivityObserver references and allows
/// subscribing or unsubscribing at runtime.
/// </summary>
public sealed class GroupActivityNotifier : IGroupActivitySubject
{
    private readonly List<IGroupActivityObserver> _observers = [];

    public int SubscriberCount => _observers.Count;

    public void Subscribe(IGroupActivityObserver observer)
    {
        ArgumentNullException.ThrowIfNull(observer);
        if (!_observers.Contains(observer))
            _observers.Add(observer);
    }

    public void Unsubscribe(IGroupActivityObserver observer)
    {
        if (observer is not null)
            _observers.Remove(observer);
    }

    public void Notify(GroupActivityEvent activity)
    {
        // Iterate over a snapshot so an observer that unsubscribes mid-loop is safe.
        foreach (var observer in _observers.ToList())
            observer.OnGroupChanged(activity);
    }
}
