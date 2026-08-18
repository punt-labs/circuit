MACHINE PRWatch
SETS
    STATE = {watch, fixing, done};
    TRANSITION = {needsWork, fixed}
VARIABLES
    current,
    checksGreen,
    reviewClean
INVARIANT
    current : STATE &
    checksGreen : BOOL &
    reviewClean : BOOL
INITIALISATION
    current := watch ||
    checksGreen := FALSE ||
    reviewClean := FALSE
OPERATIONS
    ObserveChecksGreen(result) =
        PRE
            result : BOOL
        THEN
            checksGreen := result
        END;

    ObserveReviewClean(result) =
        PRE
            result : BOOL
        THEN
            reviewClean := result
        END;

    Advance(evt) =
        PRE
            evt : TRANSITION &
            current /= done &
            (
                (current = watch & evt = needsWork) or
                (current = fixing & evt = fixed & checksGreen = TRUE & reviewClean = TRUE)
            )
        THEN
            IF current = watch & evt = needsWork THEN
                current := fixing
            ELSIF current = fixing & evt = fixed THEN
                current := done
            END
        END
END
