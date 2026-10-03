# Audit checklist

Use the sections relevant to the project and the requested audit scope.
Each item is a prompt to investigate, not a finding by itself. Check the
repository's requirements and existing conventions, then record evidence
and the effect on users or maintainers before reporting a problem.
Record the sections reviewed and any gaps in the audit's checkpoint.

## Repository conventions and integration

- [ ] Check whether the change reimplements an existing project helper.
- [ ] Check whether it builds a new HTTP client despite an established fetching library or wrapper.
- [ ] Check whether it introduces a second validator, logger, state store, or API client.
- [ ] Check whether it introduces a class into a consistently functional module, or the reverse.
- [ ] Check whether it changes global configuration where a module-level mechanism exists.
- [ ] Check whether naming, file placement, or module boundaries differ from neighboring code.
- [ ] Check whether looping, naming, or error-handling style changes without a reason.
- [ ] Check whether a generic abstraction serves only one known use case.
- [ ] Check whether compatibility branches target unsupported versions or environments.
- [ ] Check whether the change is far larger than an existing helper-based solution.
- [ ] Check whether a new function duplicates an existing function under a slightly different name.
- [ ] Check whether only new callers use the duplicate while old callers remain on the original.
- [ ] Check whether similar endpoints each implement separate validation, pagination, or response logic.
- [ ] Check whether authentication middleware differs between equivalent application areas.
- [ ] Check whether related functions are scattered across unrelated files.
- [ ] Check whether a small requested change expands into a broad refactor.
- [ ] Check whether a new dependency duplicates a capability already available.
- [ ] Check whether the patch introduces a second project-wide convention.
- [ ] Check whether a generic solution replaces a deliberate external-service workaround.
- [ ] Check whether an older compatibility path disappears without confirming its consumers.
- [ ] Check whether a new endpoint bypasses established route registration.
- [ ] Check whether a new type duplicates a domain type in neighboring modules.
- [ ] Check whether the change creates a second source of truth for one setting.
- [ ] Check whether an existing shared helper is copied and modified locally.
- [ ] Check whether similar modules now disagree on returning, throwing, or logging errors.
- [ ] Check whether the change bypasses the project's injection or factory mechanism.
- [ ] Check whether a new directory duplicates the responsibility of an existing directory.
- [ ] Check whether an internal service uses HTTP where the project uses an in-process interface.

## Abstraction and code volume

- [ ] Check whether a factory has only one construction path.
- [ ] Check whether an interface has one implementation and no defined boundary.
- [ ] Check whether a singleton wraps an object with one ordinary caller.
- [ ] Check whether a configuration system surrounds values that never vary.
- [ ] Check whether a simple operation traverses unnecessary managers, services, and adapters.
- [ ] Check whether speculative future variants displace current requirements.
- [ ] Check whether existing functionality could replace many new lines.
- [ ] Check whether a broad cleanup accompanies a surgical bug fix.
- [ ] Check whether an enum has one value yet drives switches across modules.
- [ ] Check whether a strategy map has one strategy and no defined extension contract.
- [ ] Check whether a wrapper forwards every dependency method without enforcing a policy.
- [ ] Check whether unchanged data crosses multiple layers solely to fit an architecture.
- [ ] Check whether a plugin point exists without a plugin contract or user.
- [ ] Check whether a base class contains logic for only one subclass.
- [ ] Check whether a builder constructs one fixed object shape.
- [ ] Check whether a utility exposes optional parameters that no caller uses.
- [ ] Check whether a dependency replaces only a few standard-library operations.
- [ ] Check whether removing a layer would leave behavior and the external interface unchanged.
- [ ] Check whether a shared helper combines similar-looking operations governed by different business rules.
- [ ] Check whether an abstraction accumulates caller-specific branches whenever a new use case is added.
- [ ] Check whether consumers must bypass a shared abstraction to implement ordinary supported behavior.
- [ ] Check whether changing a helper for one consumer changes unrelated consumers that should evolve independently.
- [ ] Check whether extracting a small shared package creates release coordination disproportionate to the behavior reused.
- [ ] Check whether a function is fragmented solely to meet a line-count rule, with fragments depending on shared mutable intermediates.

## Comments, documentation, and naming

- [ ] Check whether comments simply narrate the statements immediately below them.
- [ ] Check whether every small block has a heading comment without a useful reason.
- [ ] Check whether module documentation contains unsolicited features, dependencies, and usage sections.
- [ ] Check whether comments describe behavior the code does not provide.
- [ ] Check whether comments contain prompts or instructions addressed to an assistant.
- [ ] Check whether generated template or AI-use notices remain in a submitted implementation.
- [ ] Check whether internal names use generic promotional modifiers such as enhanced without distinction.
- [ ] Check whether emojis proliferate in logs or comments against project style.
- [ ] Check whether the patch's writing voice differs sharply from surrounding code.
- [ ] Check whether comments refer to stages or phases absent from repository plans.
- [ ] Check whether comments narrate what a function used to do instead of documenting current behavior.
- [ ] Check whether comments announce the next step of an implementation plan.
- [ ] Check whether file headers claim unverified features.
- [ ] Check whether a comment describes an unreachable fallback.
- [ ] Check whether a comment calls code safe without naming the threat or invariant.
- [ ] Check whether docstrings list nonexistent parameters.
- [ ] Check whether documentation examples call nonexistent classes or keys.
- [ ] Check whether TODOs refer to future work without an issue, owner, or plan.
- [ ] Check whether a comment describes the request rather than the implementation.
- [ ] Check whether identical explanatory paragraphs repeat across functions.
- [ ] Check whether a best-practices claim conflicts with project constraints.
- [ ] Check whether a changelog claims tests absent from the PR record.
- [ ] Check whether documentation examples run against the submitted interface.
- [ ] Check whether the README presents placeholder behavior as shipped.
- [ ] Check whether commented-out implementations remain alongside the active implementation after their replacement is complete.
- [ ] Check whether obsolete functions, imports, or assets remain reachable or packaged despite having no supported consumer.
- [ ] Check whether retired feature flags preserve branches that no deployed configuration can select.
- [ ] Check whether names or schema comments still describe an earlier meaning after a field has been repurposed.
- [ ] Check whether one policy value is repeated as numeric or string literals that can drift when the policy changes.

## APIs, logic, and control flow

- [ ] Check whether imports refer to nonexistent packages or modules.
- [ ] Check whether called methods or arguments exist in the installed dependency version.
- [ ] Check whether APIs from incompatible library versions are combined.
- [ ] Check whether browser APIs are used in a non-browser runtime.
- [ ] Check whether deprecated APIs appear despite modern equivalents in the repository.
- [ ] Check whether an entire function is wrapped in try/catch without a recovery decision.
- [ ] Check whether a catch block logs an error and returns apparent success.
- [ ] Check whether a catch block is empty.
- [ ] Check whether errors are converted to null where callers require a real value.
- [ ] Check whether a fallback supplies static data where failure should stop the operation.
- [ ] Check whether guaranteed methods are repeatedly checked for existence.
- [ ] Check whether optional chaining turns a broken invariant into a silent no-op.
- [ ] Check whether lifecycle flags duplicate framework lifecycle rules.
- [ ] Check whether ordinary domain invariants are absent despite handling hypothetical edge cases.
- [ ] Check whether validation leaves required state transitions incomplete.
- [ ] Check whether code changes behavior outside the requested scope.
- [ ] Check whether a workaround stems from a misunderstood dependency.
- [ ] Check whether placeholder methods or unfinished stubs are presented as complete.
- [ ] Check whether polished sections sit beside naive logic that violates local constraints.
- [ ] Check whether failure in step one is followed by a step that assumes success.
- [ ] Check whether a transaction failure leaves in-memory state updated.
- [ ] Check whether rethrown errors lose the original cause or useful context.
- [ ] Check whether retries treat every failure as transient.
- [ ] Check whether a timeout returns an empty successful result.
- [ ] Check whether parsing failure selects permissive configuration.
- [ ] Check whether an unknown enum value falls through to a valid-looking default.
- [ ] Check whether validation fails open when its validator errors.
- [ ] Check whether a bulk operation reports success after individual failures.
- [ ] Check whether a caller receives too little information to recover.
- [ ] Check whether error handling has no defined user-visible or operational result.
- [ ] Check whether a fallback masks a missing required configuration.
- [ ] Check whether an operation silently ignores an unexpected input shape.
- [ ] Check whether control flow follows a generic template rather than the domain contract.
- [ ] Check whether conditional expressions reconstruct a boolean result that the condition already supplies.
- [ ] Check whether a predicate is evaluated repeatedly despite being expensive or dependent on mutable state.
- [ ] Check whether unrelated meanings are assigned to the same local variable during one operation.
- [ ] Check whether a common operation requires callers to reconstruct the same low-level access sequence instead of using the owning module's contract.

## Backend security and business rules

- [ ] Check whether an endpoint authenticates the caller but omits per-resource authorization.
- [ ] Check whether incoming fields are trusted without project-required validation.
- [ ] Check whether equivalent routes implement authentication differently.
- [ ] Check whether token or session lifetime appears without a stated policy.
- [ ] Check whether rate limiting is attached at a level inconsistent with policy.
- [ ] Check whether payment retries account for a successful charge with a lost response.
- [ ] Check whether list queries and detail or update routes enforce the same tenant scope.
- [ ] Check whether an API accepts ownership fields supplied by the client.
- [ ] Check whether privileged operations check object permissions as well as roles.
- [ ] Check whether input validation enforces domain rules in addition to shape.
- [ ] Check whether password reset tokens are single-use.
- [ ] Check whether administrative routes use the normal authorization path.
- [ ] Check whether file upload checks include size, storage access, and processing behavior.
- [ ] Check whether secrets or connection strings are hardcoded.
- [ ] Check whether endpoints expose database objects rather than approved response shapes.
- [ ] Check whether deletion removes required related private data.
- [ ] Check whether webhook handlers verify signatures.
- [ ] Check whether logs contain credentials, tokens, or sensitive request bodies.
- [ ] Check whether a caller can query another tenant by supplying an ID.
- [ ] Check whether security defaults have an identified policy and threat model.

## Data, persistence, and concurrency

- [ ] Check whether Go request contexts propagate instead of becoming context.TODO().
- [ ] Check whether cancellation reaches downstream operations.
- [ ] Check whether Go error wrapping follows project convention.
- [ ] Check whether methods satisfy the intended Go interface on the correct type.
- [ ] Check whether individually valid Redis commands remain correct under concurrency.
- [ ] Check whether database tests exercise the production database's relevant behavior.
- [ ] Check whether read-modify-write logic needs concurrency protection.
- [ ] Check whether upserts preserve immutable fields.
- [ ] Check whether retries repeat non-idempotent writes.
- [ ] Check whether success is acknowledged before a durable write completes.
- [ ] Check whether pagination uses a stable sort order.
- [ ] Check whether duplicate event delivery has a defined result.
- [ ] Check whether all write paths invalidate or update related caches.
- [ ] Check whether asynchronous tasks outlive captured request-scoped objects.
- [ ] Check whether event listeners are removed on teardown.
- [ ] Check whether database constraints enforce critical domain invariants.
- [ ] Check whether a migration blocks while rewriting a large table.
- [ ] Check whether a migration supports old and new application versions during rollout.
- [ ] Check whether external calls inside transactions have a recovery plan.
- [ ] Check whether partial batch updates can be identified and resumed.

## Tests and verification

- [ ] Check whether a large test diff consists mostly of repeated setup or low-value assertions.
- [ ] Check whether tests assert only what their mocks were configured to return.
- [ ] Check whether expected results independently reflect requirements rather than copied implementation logic.
- [ ] Check whether tests exercise failure paths and real domain edge cases.
- [ ] Check whether contrived edge cases displace known production cases.
- [ ] Check whether mocks replace the integration that the change must prove.
- [ ] Check whether user-visible paths have integration or end-to-end coverage when required.
- [ ] Check whether tests cover the exact state named in a requirement.
- [ ] Check whether implementation and tests share a mistaken assumption.
- [ ] Check whether a test still passes when the relevant production function is replaced by a no-op.
- [ ] Check whether mocks always return success.
- [ ] Check whether authorization tests access another user's resource.
- [ ] Check whether concurrency tests actually overlap operations.
- [ ] Check whether integration tests mock both sides of the boundary.
- [ ] Check whether browser tests complete the user's task rather than check element presence.
- [ ] Check whether snapshots bless placeholder content or invalid semantics.
- [ ] Check whether tests call the public route or component rather than only new helpers.
- [ ] Check whether changed tests follow an explicitly changed requirement.
- [ ] Check whether external API mocks use documented or recorded responses.
- [ ] Check whether local tests use a materially different database or queue.
- [ ] Check whether each test name matches the case its inputs create.
- [ ] Check whether test claims identify the risky behavior exercised.
- [ ] Check whether coverage targets add tests that provide new evidence.
- [ ] Check whether tests pass only because exceptions are swallowed.
- [ ] Check whether test data resembles real content lengths, permissions, and failure conditions.
- [ ] Check whether tests fail after an internal refactor that preserves the supported behavior and interface.
- [ ] Check whether tests assert private helper calls or call order that the public contract does not require.
- [ ] Check whether production visibility is widened solely to let tests invoke implementation details.
- [ ] Check whether mocks prevent a behavior-preserving implementation from replacing one internal algorithm with another.
- [ ] Check whether disabled or skipped tests silently remove coverage for behavior still claimed as supported.
- [ ] Check whether the build reports success despite failing tests or analysis steps being marked as allowed failures.

## Frontend implementation and interaction

- [ ] Check whether decorative buttons, filters, search, and forms actually complete a task.
- [ ] Check whether controls support keyboard use and visible focus.
- [ ] Check whether images and interactive elements have meaningful labels.
- [ ] Check whether responsive layouts work at narrow widths.
- [ ] Check whether longer or translated text remains usable.
- [ ] Check whether React effects derive values already available during rendering.
- [ ] Check whether a converted lifecycle method triggers an effect on every render.
- [ ] Check whether effects form an avoidable state-update chain.
- [ ] Check whether empty states are designed.
- [ ] Check whether error states offer a useful next action.
- [ ] Check whether contrast is sufficient despite a polished palette.
- [ ] Check whether a search field actually filters or submits.
- [ ] Check whether a success message waits for server confirmation.
- [ ] Check whether modals can be dismissed by keyboard.
- [ ] Check whether loading states eventually communicate errors.
- [ ] Check whether duplicated local and server state has a synchronization rule.
- [ ] Check whether effect dependencies include every value read.
- [ ] Check whether unstable dependencies cause repeated fetching.
- [ ] Check whether stale requests can overwrite newer results.
- [ ] Check whether clickable div elements need semantic buttons.
- [ ] Check whether labels are programmatically associated with inputs.
- [ ] Check whether disabled controls explain why they are disabled.
- [ ] Check whether optimistic UI restores state after persistence fails.
- [ ] Check whether hover effects exist without corresponding keyboard behavior.
- [ ] Check whether controls appear interactive but have placeholder handlers.

## Visual design and page content

- [ ] Check whether a generic SaaS hero appears without a product-specific design brief.
- [ ] Check whether blue or purple gradients ignore established branding.
- [ ] Check whether glass effects decorate surfaces without a layering need.
- [ ] Check whether a tiny New or sparkle badge sits above every centered hero.
- [ ] Check whether the hero follows a badge–headline–subhead–two-button–glow template.
- [ ] Check whether abstract gradient orbs serve no communicative purpose.
- [ ] Check whether features are forced into exactly three equal cards.
- [ ] Check whether every feature card starts with an icon in a rounded square.
- [ ] Check whether a bento grid appears without a content reason.
- [ ] Check whether all sections use nearly identical spacing.
- [ ] Check whether one large corner radius is applied to every component.
- [ ] Check whether cards are nested where information hierarchy would suffice.
- [ ] Check whether ordinary cards use colored alert-like borders decoratively.
- [ ] Check whether client logos represent real relationships.
- [ ] Check whether animated usage numbers have a verifiable basis.
- [ ] Check whether compliance badges reflect completed audits.
- [ ] Check whether testimonials name verifiable customers and specific use cases.
- [ ] Check whether the page follows a full hero–logos–features–pricing–FAQ template without need.
- [ ] Check whether every section repeats the same fade-up animation.
- [ ] Check whether essential state transitions lack motion while decorative cards bounce.
- [ ] Check whether a fake terminal hero fits the product.
- [ ] Check whether the favicon or generator attribution remains from a starter.
- [ ] Check whether an italic serif accent word fits the product voice.
- [ ] Check whether generic lines such as Transform your workflow replace specific copy.
- [ ] Check whether an untouched component-library theme clashes with existing design tokens.
- [ ] Check whether neon accents on dark backgrounds serve the brand.
- [ ] Check whether gradient text is used on metrics or headings without hierarchy benefit.
- [ ] Check whether mono type is decorative on a nontechnical page.
- [ ] Check whether one font family and weight flatten typography hierarchy.
- [ ] Check whether icon choices repeat generic sparkle, zap, shield, and chart symbols.

## Infrastructure, deployment, and operations

- [ ] Check whether Terraform plans fit the organization's actual environment.
- [ ] Check whether internal Terraform modules already supply the capability.
- [ ] Check whether each new resource has a clear owner and deletion path.
- [ ] Check whether IAM permissions exceed the operation's requirements.
- [ ] Check whether a workload bypasses an established platform CRD or abstraction.
- [ ] Check whether namespaces, storage classes, ingress classes, and service accounts exist in each target cluster.
- [ ] Check whether reusable modules contain environment-specific literals.
- [ ] Check whether Terraform creates a resource that should be referenced or imported.
- [ ] Check whether dependency ordering handles replacement and deletion as well as creation.
- [ ] Check whether network policies or security groups allow only required traffic.
- [ ] Check whether readiness probes wait for actual serving readiness.
- [ ] Check whether liveness probes tolerate recoverable downstream outages.
- [ ] Check whether rollouts work while old and new versions coexist.
- [ ] Check whether the artifact tested in CI is the artifact deployed.
- [ ] Check whether pipeline output, rendered manifests, or logs expose secrets.
- [ ] Check whether a Helm value is consumed by a template.
- [ ] Check whether alerts refer to metrics and labels emitted in deployment.
- [ ] Check whether retries amplify failures of a dependency.
- [ ] Check whether new operational complexity has failure signals and rollback steps.
- [ ] Check whether temporary console output bypasses configured log levels, formatting, or redaction.
- [ ] Check whether one propagated failure produces duplicate error records at several layers without additional context.
- [ ] Check whether routine successful operations are logged as warnings or errors and pollute failure signals.
- [ ] Check whether concurrent-operation logs omit the correlation fields required to associate events with their operation.
- [ ] Check whether the same event uses inconsistent field names or types across its logging paths.
- [ ] Check whether logging eagerly computes expensive payloads even when the selected log level is disabled.

## Change scope and completeness

- [ ] Check whether domain decisions violate project requirements or constraints.
- [ ] Check whether the change has a testable statement of required behavior.
- [ ] Check whether the PR scope matches the requested outcome.
- [ ] Check whether the submission claims completeness while leaving stubs.

## Dependency structure and change propagation

- [ ] Check whether module dependency cycles prevent independent replacement or testing.
- [ ] Check whether one business-rule change requires coordinated edits across unrelated modules.
- [ ] Check whether one module changes for unrelated persistence, presentation, and business-policy requirements.
- [ ] Check whether domain rules depend directly on framework, transport, or database implementation types.
- [ ] Check whether consumers import another module's internal files instead of its supported interface.
- [ ] Check whether a shared utility package depends on the application modules that consume it.
- [ ] Check whether unrelated features share a mutable object that couples their behavior.
- [ ] Check whether a service boundary requires a synchronous call back to its caller to complete ordinary work.
- [ ] Check whether supposedly independent services require coordinated releases for routine interface changes.
- [ ] Check whether multiple services write the same tables without exclusive ownership of their invariants.
- [ ] Check whether adding one variant requires modifying the same dispatch logic in several modules.
- [ ] Check whether a dependency replacement requires business-rule changes outside its adapter.
- [ ] Check whether domain code resolves dependencies from a global service locator instead of receiving its required capabilities.
- [ ] Check whether missing service registrations remain undetected until a rarely executed branch runs.
- [ ] Check whether a component needs numerous unrelated dependencies because it combines separate responsibilities.
- [ ] Check whether tests must populate an application-wide container to construct one otherwise isolated component.

## Domain types and invalid states

- [ ] Check whether one type permits combinations of fields that the domain forbids.
- [ ] Check whether several booleans encode mutually exclusive lifecycle states.
- [ ] Check whether separate identifiers with different meanings use interchangeable primitive types.
- [ ] Check whether quantities cross interfaces without an enforced unit or conversion boundary.
- [ ] Check whether money calculations use binary floating-point where exact decimal rules are required.
- [ ] Check whether absence, zero, empty content, and failure share one representation despite different behavior.
- [ ] Check whether constructors return objects before required invariants are established.
- [ ] Check whether public setters allow state transitions that bypass domain validation.
- [ ] Check whether callers must repeat validation before every use of an allegedly validated value.
- [ ] Check whether string concatenation creates structured identifiers without parsing or escaping rules.
- [ ] Check whether timestamps lose required timezone or offset information at storage boundaries.
- [ ] Check whether elapsed-time measurements depend on a wall clock that can jump.
- [ ] Check whether serialization silently truncates values or changes their precision.
- [ ] Check whether ordering or equality omits fields required by the domain contract.

## Function contracts and hidden effects

- [ ] Check whether an operation advertised as a query modifies persistent or shared state.
- [ ] Check whether a function mutates caller-owned arguments without an explicit mutation contract.
- [ ] Check whether correctness depends on an undocumented order of initialization calls.
- [ ] Check whether a function's result depends on mutable globals absent from its interface.
- [ ] Check whether callers must set temporary global configuration before invoking a function.
- [ ] Check whether boolean mode arguments select unrelated operations with different preconditions.
- [ ] Check whether parameter combinations permit unsupported operating modes that fail only deep in execution.
- [ ] Check whether a function accepts a broad object but requires only a small capability from it.
- [ ] Check whether returned collections expose mutable internal state and permit invariant violations.
- [ ] Check whether a base implementation invokes overridable methods before subclass initialization completes.
- [ ] Check whether a subtype rejects inputs or omits guarantees accepted by its public base contract.
- [ ] Check whether a method requires runtime type checks for every supported implementation of an interface.
- [ ] Check whether callbacks can re-enter an operation while its state is temporarily inconsistent.
- [ ] Check whether a failed operation leaves externally visible mutations without a defined failure guarantee.
- [ ] Check whether Python mutable default arguments retain data across calls that require independent state.
- [ ] Check whether objects share one mutable class-level collection despite requiring per-instance state.
- [ ] Check whether configuration readers add, delete, or rewrite keys in a shared configuration dictionary.
- [ ] Check whether a custom configuration language permits control flow or executable expressions without corresponding validation and test support.
- [ ] Check whether C# asynchronous operations use async void outside event-handler requirements and prevent callers from awaiting completion or observing failure.

## Resource ownership and asynchronous lifetime

- [ ] Check whether more than one component assumes responsibility for closing the same resource.
- [ ] Check whether resources are released on exceptions and early returns as well as successful completion.
- [ ] Check whether a resource can escape the scope that owns its lifetime.
- [ ] Check whether background tasks are launched without a component that observes their completion and failures.
- [ ] Check whether shutdown waits indefinitely for tasks that have no termination path.
- [ ] Check whether timers or scheduled jobs continue after their owning component is disposed.
- [ ] Check whether reconnecting creates duplicate workers or subscriptions before old ones terminate.
- [ ] Check whether locks remain held during external I/O or callbacks into unknown code.
- [ ] Check whether multiple locks can be acquired in inconsistent orders.
- [ ] Check whether asynchronous code blocks the event loop or exhausts a worker pool with synchronous waits.
- [ ] Check whether a task waits for work scheduled onto the same exhausted executor.
- [ ] Check whether concurrency limits apply separately to callers while leaving aggregate work unbounded.
- [ ] Check whether object reuse exposes stale request data to another request or tenant.
- [ ] Check whether cleanup failures replace the original operation failure and discard its cause.

## Work bounds and overload behavior

- [ ] Check whether pending work can accumulate without a queue limit or admission policy.
- [ ] Check whether queue age can exceed request deadlines while expired work still executes.
- [ ] Check whether a slow consumer can retain unlimited messages or block unrelated consumers.
- [ ] Check whether a fan-out operation creates one task per input without a concurrency bound.
- [ ] Check whether downstream calls can wait indefinitely because no timeout is enforced.
- [ ] Check whether each nested call starts a fresh timeout instead of respecting the remaining operation deadline.
- [ ] Check whether retries at multiple call-chain layers multiply attempts for one logical operation.
- [ ] Check whether synchronized retries or scheduled jobs create recurring load spikes.
- [ ] Check whether retries continue after the caller's deadline has expired.
- [ ] Check whether capacity controls bound item count while permitting unbounded item sizes.
- [ ] Check whether caches grow indefinitely or retain entries after their useful lifetime.
- [ ] Check whether request-controlled dimensions permit excessive CPU, memory, or recursion depth.
- [ ] Check whether one expensive request can monopolize resources required by ordinary requests.
- [ ] Check whether an overloaded instance continues accepting work it cannot finish within its latency target.

## Data access and algorithmic scaling

- [ ] Check whether fetching a collection performs an additional query for each returned item.
- [ ] Check whether one logical remote operation requires avoidable round trips for individual fields.
- [ ] Check whether filtering or aggregation loads an entire dataset when the datastore can perform it.
- [ ] Check whether a response has no enforced result-size limit or continuation mechanism.
- [ ] Check whether routine work repeatedly scans a collection that grows with total system history.
- [ ] Check whether nested scans produce quadratic work at expected production input sizes.
- [ ] Check whether a database query lacks an index aligned with its filtering and ordering requirements.
- [ ] Check whether serialization or logging repeatedly copies large payloads on the critical path.
- [ ] Check whether a batch merely moves unbounded memory usage from the database into the application.
- [ ] Check whether independent I/O operations are serialized and exceed the operation's latency budget.
- [ ] Check whether optimization introduces cached derived state without accounting for its update costs.
- [ ] Check whether performance tests use dataset sizes too small to expose the chosen algorithm's growth.

## Configuration and verification isolation

- [ ] Check whether a configuration change requires rebuilding code despite being specific to a deployment.
- [ ] Check whether invalid configuration is accepted at startup and fails only on the first affected request.
- [ ] Check whether configuration combinations bypass validation that individual settings pass.
- [ ] Check whether import-time initialization opens connections or starts workers before configuration is validated.
- [ ] Check whether tests require a live external service to exercise otherwise deterministic domain rules.
- [ ] Check whether tests depend on execution order or state left by another test.
- [ ] Check whether tests mutate process-wide settings without restoring them.
- [ ] Check whether timing-sensitive tests use fixed sleeps instead of observing the required condition.
- [ ] Check whether time and randomness are uncontrolled inputs in tests that require reproducible results.
- [ ] Check whether concurrent tests share files, ports, or records without isolation.
- [ ] Check whether test cleanup leaves persistent records or workers that affect later runs.
- [ ] Check whether compiler, type-checker, or static-analysis failures are suppressed across entire modules to accommodate a local defect.

## Alerting and service objectives

- [ ] Check whether a paging alert has no immediate mitigation or escalation action.
- [ ] Check whether resource-utilization thresholds page without evidence of service impact or imminent exhaustion.
- [ ] Check whether alert severity fails to distinguish urgent intervention from work that can wait.
- [ ] Check whether one dependency failure pages separately for every affected instance without grouping or inhibition.
- [ ] Check whether brief normal fluctuations repeatedly trigger and resolve the same alert.
- [ ] Check whether paging rules omit a valid owning service and routing destination.
- [ ] Check whether alerts link to missing or obsolete recovery procedures.
- [ ] Check whether missing telemetry makes an alert silently evaluate as healthy.
- [ ] Check whether an SLI excludes failures or slow requests that belong in its service objective.
- [ ] Check whether the alert detection window leaves insufficient time to intervene before the error budget is exhausted.

## Metrics and operational evidence

- [ ] Check whether metric labels contain unbounded request IDs, user IDs, or raw URL paths.
- [ ] Check whether a metric's name or unit disagrees with the value it records.
- [ ] Check whether latency dashboards show averages while omitting the tail required by the service objective.
- [ ] Check whether success metrics count accepted work instead of completed work.
- [ ] Check whether asynchronous work loses trace context at queue or worker boundaries.
- [ ] Check whether a critical workflow lacks metrics for its failures, duration, or pending work.
- [ ] Check whether logs are stored only in an ephemeral container filesystem and disappear on replacement.
- [ ] Check whether telemetry failure blocks the application's serving path without a bounded failure policy.

## Deployment ordering and release identity

- [ ] Check whether production deployment depends on scripts or files absent from version control.
- [ ] Check whether mutable image tags permit the same declared release to resolve to different image contents.
- [ ] Check whether a deployed binary lacks an accessible revision or build identifier.
- [ ] Check whether an older pipeline can deploy after a newer pipeline and overwrite the intended release.
- [ ] Check whether simultaneous deployments to one environment can interleave incompatible steps.
- [ ] Check whether a deployment is marked successful before application health and rollout completion are verified.
- [ ] Check whether rollback requires an artifact that has already been deleted or overwritten.
- [ ] Check whether a build depends on undeclared tools or files left on a persistent runner.

## Desired state and infrastructure ownership

- [ ] Check whether routine production changes occur outside the authoritative infrastructure or configuration source.
- [ ] Check whether emergency changes remain unreconciled and are overwritten by the next deployment.
- [ ] Check whether two controllers continuously overwrite the same resource fields.
- [ ] Check whether broad drift exclusions hide fields the project is responsible for managing.
- [ ] Check whether removed resources persist because deletion behavior is absent from the reconciliation policy.
- [ ] Check whether automation requires manual copying of resource identifiers already available as provisioning outputs.
- [ ] Check whether repeated reconciliation changes an already converged system without an input change.
- [ ] Check whether one environment is configured through undocumented exceptions that cannot be recreated from its declared state.

## Terraform state and change safety

- [ ] Check whether concurrent Terraform writers can modify the same state without supported locking.
- [ ] Check whether automation disables state locking to bypass contention.
- [ ] Check whether force-unlock can run while another state-writing operation is still active.
- [ ] Check whether unrelated infrastructure shares a state boundary that unnecessarily expands the impact of a change.
- [ ] Check whether broad ignore_changes rules conceal required configuration updates.
- [ ] Check whether routine applies depend on resource targeting that leaves the full configuration unconverged.

## Automation failure and rerun behavior

- [ ] Check whether a deployment script continues after a required command fails.
- [ ] Check whether Ansible ignores task failures without verifying an acceptable alternate outcome.
- [ ] Check whether Ansible command tasks report changes on every run despite making no changes.
- [ ] Check whether configuration updates leave services using old settings when notified handlers never execute.
- [ ] Check whether a failed automation run cannot be restarted without duplicating completed actions.
- [ ] Check whether routine remediation requires repeated manual database edits, file cleanup, or process restarts.

## Container lifecycle and scheduling

- [ ] Check whether the container entrypoint prevents the application from receiving termination signals.
- [ ] Check whether terminating instances continue accepting new work instead of draining existing work.
- [ ] Check whether the configured termination grace period is shorter than the supported drain operation.
- [ ] Check whether every replica performs the same database migration at startup without coordination.
- [ ] Check whether resource requests omit the capacity needed for realistic steady-state operation.
- [ ] Check whether configured memory limits cause repeatable termination during supported workloads.
- [ ] Check whether replicas advertised as redundant all depend on one host or failure domain.
- [ ] Check whether application persistence relies on ephemeral container storage despite requiring survival across replacements.

## Backup and disaster recovery

- [ ] Check whether backup success is accepted without testing restoration and application-level data validity.
- [ ] Check whether recovery exercises restore data but omit infrastructure, identity, secrets, or network dependencies.
- [ ] Check whether measured recovery time exceeds the service's recovery-time objective.
- [ ] Check whether backup frequency and retained recovery points cannot meet the recovery-point objective.
- [ ] Check whether backups share a failure or deletion boundary with the primary data they protect.
- [ ] Check whether restoring encrypted backups depends on unavailable or expired decryption keys.
- [ ] Check whether recovery tooling requires the same unavailable infrastructure it is intended to restore.
- [ ] Check whether a restored system passes storage checks but fails the application's critical workflows.

## Security boundaries and injection resistance

- [ ] Check whether hiding a UI control substitutes for enforcing the corresponding permission on the server.
- [ ] Check whether alternate entry points, exports, or background jobs bypass the authorization used by the primary API.
- [ ] Check whether permission revocation leaves cached authorization decisions effective beyond the permitted interval.
- [ ] Check whether static downloads or generated reports bypass the protection applied to their source records.
- [ ] Check whether SQL statements concatenate untrusted values instead of using parameter binding.
- [ ] Check whether dynamic table names, column names, or sort expressions accept arbitrary client input instead of approved mappings.
- [ ] Check whether server-side URL fetching permits access to internal services or cloud metadata endpoints.
- [ ] Check whether redirects can move an initially approved outbound request to a forbidden destination.
- [ ] Check whether hostname validation and subsequent connection handling permit DNS resolution to bypass destination restrictions.
- [ ] Check whether outbound TLS clients disable certificate or hostname verification to work around configuration errors.
- [ ] Check whether token validation omits the required issuer, audience, expiration, or signature checks.
- [ ] Check whether token verification accepts an algorithm selected solely by untrusted token metadata.
- [ ] Check whether CORS configuration is treated as an authentication or authorization mechanism.
- [ ] Check whether untrusted content reaches HTML insertion APIs that bypass framework escaping.
- [ ] Check whether output encoding is applied for the wrong destination context, such as HTML encoding inside JavaScript.
- [ ] Check whether sanitized HTML is modified afterward in ways that reintroduce executable content.
- [ ] Check whether arbitrary URL schemes can enter links or redirects intended to allow only safe destinations.
- [ ] Check whether an application exposes debug endpoints or detailed internal errors in production responses.

## Database modeling and transaction behavior

- [ ] Check whether records requiring stable identity lack a primary key or equivalent enforced identifier.
- [ ] Check whether a uniqueness check uses a preliminary query without protection against concurrent inserts.
- [ ] Check whether related records can become orphaned because referential integrity is absent from all write paths.
- [ ] Check whether mandatory columns accept NULL values that the application cannot process.
- [ ] Check whether uniqueness requirements account for the database's treatment of NULL values.
- [ ] Check whether multi-valued relationships are stored as delimiter-separated strings that prevent required relational operations.
- [ ] Check whether denormalized values can disagree because their write paths have no enforced consistency rule.
- [ ] Check whether historical reports change retroactively when mutable reference records are updated.
- [ ] Check whether cascading deletion can remove substantially more data than the operation is authorized or intended to delete.
- [ ] Check whether a long-lived idle transaction retains locks or prevents necessary database maintenance.
- [ ] Check whether transaction isolation is weaker than the business rule's required consistency guarantee.
- [ ] Check whether a deadlock retry repeats only one statement when the complete transaction must be retried.
- [ ] Check whether schema changes require stronger locks than the deployment can tolerate.
- [ ] Check whether query-plan estimates differ substantially from actual row counts without investigation of statistics or selectivity.
- [ ] Check whether representative parameter values produce materially different plans from the values used in testing.
- [ ] Check whether functions or implicit conversions on indexed predicates prevent the intended access path.
- [ ] Check whether large updates or deletions create sustained table bloat or maintenance pressure beyond available capacity.
- [ ] Check whether database connection pools across all replicas exceed the database's usable connection budget.

## QA coverage and trustworthy test results

- [ ] Check whether rerunning failed tests until they pass conceals the initial failure in the release result.
- [ ] Check whether quarantine removes unreliable tests without another check for the behavior they protected.
- [ ] Check whether one oversized end-to-end flow prevents later assertions from executing when an earlier unrelated step fails.
- [ ] Check whether deterministic business rules are verified only through slow browser journeys despite an available lower-level interface.
- [ ] Check whether browser selectors depend on incidental DOM nesting or styling classes that are outside the tested contract.
- [ ] Check whether a browser test forces an action that would fail normal visibility or interaction checks.
- [ ] Check whether asynchronous assertions sample state once instead of waiting for the required condition within a defined timeout.
- [ ] Check whether test fixtures silently assume records or accounts already exist in a shared environment.
- [ ] Check whether broad exception handling in test helpers converts assertion failures into successful completion.
- [ ] Check whether a test runner exits successfully after discovering no required tests.
- [ ] Check whether parameterized tests reuse one mutable fixture and contaminate subsequent cases.
- [ ] Check whether tests cover valid boundaries and the immediately adjacent invalid values.
- [ ] Check whether tests omit interrupted, repeated, or out-of-order steps in workflows that permit those conditions.
- [ ] Check whether tests verify persisted results after a successful response rather than trusting the response alone.
- [ ] Check whether a bug fix lacks a regression case that fails against the affected behavior.
- [ ] Check whether contract tests omit incompatible response fields or status codes that consumers depend on.
- [ ] Check whether tests assume an exact order for results whose interface provides no ordering guarantee.
- [ ] Check whether failure output omits the inputs, environment, or captured evidence needed to reproduce the failing case.

## Documentation concision and substance

- [ ] Check whether an introduction repeats the title instead of stating the document's purpose and scope.
- [ ] Check whether paragraphs restate the same claim without adding a condition, example, constraint, or action.
- [ ] Check whether a summary repeats a short section that already contains the complete information.
- [ ] Check whether generic explanations of standard technology displace the project's actual configuration and constraints.
- [ ] Check whether instructions include filler steps that do not change state or verify an outcome.
- [ ] Check whether lengthy feature descriptions omit the limitations that determine whether the feature is usable.
- [ ] Check whether vague advice such as following best practices substitutes for a specific project requirement.
- [ ] Check whether a document expands a small change into unrelated tutorials or speculative future work.
- [ ] Check whether multiple documents repeat the same instructions and create independently maintained copies.
- [ ] Check whether a documentation file has no distinct purpose beyond restating existing material.

## Documentation tone and generated prose patterns

- [ ] Check whether descriptions use promotional claims such as robust, seamless, or comprehensive without concrete supporting properties.
- [ ] Check whether security, reliability, or production-readiness claims exceed the evidence available for the documented release.
- [ ] Check whether repeated contrast formulas such as not X but Y introduce distinctions that the subject does not require.
- [ ] Check whether abstract metaphors or invented compound terms replace established technical names.
- [ ] Check whether strings of positive adjectives replace measurable behavior or explicit guarantees.
- [ ] Check whether canned transitions recur without expressing a real logical relationship between statements.
- [ ] Check whether conversational phrases such as great question or I hope this helps remain in permanent documentation.
- [ ] Check whether a document addresses a prompting user or offers follow-up assistance instead of documenting the project.
- [ ] Check whether generated prose uses certainty where the source material identifies an assumption or unresolved question.
- [ ] Check whether terminology changes between sections for the same component, operation, or state.

## Documentation structure and formatting

- [ ] Check whether nearly every sentence is bolded and emphasis no longer distinguishes warnings or key terms.
- [ ] Check whether headings divide individual sentences into sections without distinct topics.
- [ ] Check whether deeply nested bullets fragment one procedure that requires an explicit sequence.
- [ ] Check whether tables contain long prose without a meaningful comparison or mapping between columns.
- [ ] Check whether decorative emojis, badges, or callouts conflict with the project's documentation style.
- [ ] Check whether generic headings such as key takeaways replace names of the actual operation or constraint.
- [ ] Check whether required prerequisites appear after the commands that depend on them.
- [ ] Check whether explanatory material interrupts a procedure between dependent steps.
- [ ] Check whether diagrams contain broken labels, incorrect connections, or components absent from the system.
- [ ] Check whether a quick-start section requires searching later sections for mandatory configuration before it can work.

## Documentation evidence and executable procedures

- [ ] Check whether architectural rationale is inferred from code and presented as an established design decision without a supporting record.
- [ ] Check whether precise numbers, status codes, limits, or timings lack support in the implementation or authoritative reference.
- [ ] Check whether citations exist but do not support the nearby claim or apply to the documented version.
- [ ] Check whether examples combine syntax, defaults, or options from incompatible product versions.
- [ ] Check whether sample commands require undeclared files, permissions, tools, or working-directory assumptions.
- [ ] Check whether placeholders resemble usable values and are passed into commands without substitution instructions.
- [ ] Check whether a procedure omits an expected result or a check that confirms the operation succeeded.
- [ ] Check whether destructive instructions omit their scope, required safeguards, or recovery implications.
- [ ] Check whether documentation presents proposed architecture or planned behavior as currently implemented.
- [ ] Check whether generated change summaries assert guarantees that were not validated by the change.

## Schema fidelity and domain modeling

- [ ] Check whether generated relationships or column constraints exist only in model code and are absent from the deployed schema.
- [ ] Check whether ORM models, API schemas, and generated client types disagree on field names, types, or requiredness.
- [ ] Check whether independently generated copies of a contract drift despite an available authoritative schema and deterministic generator.
- [ ] Check whether a schema change silently replaces an established domain relationship with a conventional but incompatible alternative.
- [ ] Check whether relationship cardinality permits multiple records where exactly one is required, or prevents multiple records where they are valid.
- [ ] Check whether many-to-many associations lose attributes that belong to the relationship itself, such as role, quantity, or effective dates.
- [ ] Check whether polymorphic references combine a type string and an ID without enforcing which target records are valid.
- [ ] Check whether foreign keys permit a child record to reference a parent in another tenant despite requiring tenant-local ownership.
- [ ] Check whether generic entity or attribute tables replace concrete domain records without a requirement for runtime-defined entities.
- [ ] Check whether core queryable or constrained fields are buried in an unvalidated metadata object without an equivalent enforcement mechanism.
- [ ] Check whether repeated columns such as item1, item2, and item3 impose an arbitrary limit on a variable-length relationship.
- [ ] Check whether per-record tables require schema changes and cross-table scans for ordinary record creation and retrieval.
- [ ] Check whether stored age, elapsed duration, or other time-dependent derived values require scheduled rewrites when stable source facts would suffice.

## Schema validation semantics

- [ ] Check whether mandatory JSON properties are declared under properties but omitted from required.
- [ ] Check whether a JSON schema accepts unexpected fields despite an interface contract that requires their rejection.
- [ ] Check whether JSON Schema default annotations are assumed to populate missing values without a separate defaulting operation.
- [ ] Check whether JSON Schema readOnly or writeOnly annotations are assumed to enforce request permissions or response filtering.
- [ ] Check whether string format declarations are relied on for rejection without verifying that the deployed validator asserts those formats.
- [ ] Check whether schema composition combines allOf and additionalProperties in a way that rejects intended extension fields.
- [ ] Check whether overlapping oneOf alternatives reject valid records that match more than one branch.
- [ ] Check whether incompatible constraints make a schema branch impossible to satisfy.
- [ ] Check whether schema keywords or reference behavior belong to a different dialect than the configured validator supports.
- [ ] Check whether external or local schema references fail to resolve in the packaged application or offline validation environment.
- [ ] Check whether a schema is weakened solely to accept incorrect generated payloads rather than implementing the required contract.

## Data structure shape and evolution

- [ ] Check whether parallel arrays rely on matching positions to associate fields that belong in one record.
- [ ] Check whether array positions serve as persistent identities even though insertion, deletion, or sorting changes those positions.
- [ ] Check whether converting records into a map silently overwrites valid entries with duplicate keys.
- [ ] Check whether a set removes duplicates that have distinct meaning in the domain, or a list permits duplicates that violate its membership rules.
- [ ] Check whether dictionary keys encode business fields that need separate validation, indexing, or updates.
- [ ] Check whether a tagged variant's discriminator can disagree with the fields actually present in its payload.
- [ ] Check whether one object mixes editable input fields with server-owned fields and permits unintended writes through generic mapping.
- [ ] Check whether serialize-and-parse round trips discard fields required for forwarding or later processing.
- [ ] Check whether new enum values break existing consumers that must support forward-compatible data exchange.
- [ ] Check whether Protocol Buffers field numbers are renumbered or reused after messages have entered use.
- [ ] Check whether deleted Protocol Buffers fields leave their numbers unreserved and available for accidental reuse.
- [ ] Check whether records persisted under an older structure lack a supported decoding or migration path after the structure changes.
