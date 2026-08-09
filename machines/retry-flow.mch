MACHINE RetryFlow
SETS
    STATE = {waiting, done};
    TRANSITION = {proceed}
VARIABLES
    current,
    gateOpen
INVARIANT
    current : STATE &
    gateOpen : BOOL
INITIALISATION
    current := waiting ||
    gateOpen := FALSE
OPERATIONS
    ObserveGateOpen(result) =
        PRE
            result : BOOL
        THEN
            gateOpen := result
        END;

    Advance(evt) =
        PRE
            evt : TRANSITION &
            current /= done &
            current = waiting &
            evt = proceed &
            gateOpen = TRUE
        THEN
            current := done
        END
END
