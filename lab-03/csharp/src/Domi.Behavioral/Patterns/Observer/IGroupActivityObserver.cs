namespace Domi.Behavioral.Patterns.Observer;

// ===========================================================================
// OBSERVER
// Subject interface:  IGroupActivitySubject
// ConcreteSubject:    GroupActivityNotifier (GroupActivityNotifier.cs)
// Observer interface: IGroupActivityObserver
// ConcreteObservers:  ActivityLogObserver, FullGroupAlertObserver,
//                     ActivityHistoryObserver (ActivityObservers.cs)
// ===========================================================================

/// <summary>Public snapshot sent to observers after a group state change.</summary>
public sealed record GroupActivityEvent(
    string Action,
    string GroupId,
    string GroupName,
    int MemberCount,
    int Capacity,
    string? ProfileId = null);

/// <summary>OBSERVER: reacts to group changes without coupling the subject to concrete classes.</summary>
public interface IGroupActivityObserver
{
    void OnGroupChanged(GroupActivityEvent activity);
}

/// <summary>SUBJECT: manages runtime subscriptions and broadcasts changes.</summary>
public interface IGroupActivitySubject
{
    void Subscribe(IGroupActivityObserver observer);
    void Unsubscribe(IGroupActivityObserver observer);
    void Notify(GroupActivityEvent activity);
}
