MACHINE ReviewFlow
SETS
    STATE = {coding, codeReview, done};
    TRANSITION = {requestReview, approve}
VARIABLES
    current,
    makeCheckPassed
INVARIANT
    current : STATE &
    makeCheckPassed : BOOL
INITIALISATION
    current := coding ||
    makeCheckPassed := FALSE
OPERATIONS
    ObserveMakeCheckPassed(result) =
        PRE
            result : BOOL
        THEN
            makeCheckPassed := result
        END;

    Advance(evt) =
        PRE
            evt : TRANSITION &
            current /= done &
            (
                (current = coding & evt = requestReview & makeCheckPassed = TRUE) or
                (current = codeReview & evt = approve)
            )
        THEN
            IF current = coding & evt = requestReview THEN
                current := codeReview
            ELSIF current = codeReview & evt = approve THEN
                current := done
            END
        END
END
