MACHINE PRWatch
SETS
    STATE = {watch, fixing, done};
    TRANSITION = {needsWork, fixed}
VARIABLES
    current
INVARIANT
    current : STATE
INITIALISATION
    current := watch
OPERATIONS
    Advance(evt) =
        PRE
            evt : TRANSITION &
            current /= done &
            (
                (current = watch & evt = needsWork) or
                (current = fixing & evt = fixed)
            )
        THEN
            IF current = watch & evt = needsWork THEN
                current := fixing
            ELSIF current = fixing & evt = fixed THEN
                current := done
            END
        END
END
