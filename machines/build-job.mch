MACHINE BuildJob
SETS
    STATE = {idle, running, done};
    TRANSITION = {start, finish}
VARIABLES
    current
INVARIANT
    current : STATE
INITIALISATION
    current := idle
OPERATIONS
    Advance(evt) =
        PRE
            evt : TRANSITION &
            current /= done &
            (
                (current = idle & evt = start) or
                (current = running & evt = finish)
            )
        THEN
            IF current = idle & evt = start THEN
                current := running
            ELSIF current = running & evt = finish THEN
                current := done
            END
        END
END
