<?php
// Bootstrap CloudPoolManager du Moodle local — exécuté DANS le conteneur via l'API Moodle.
// Idempotent : relançable sans dupliquer. Affiche le token WS sur stdout (ligne "TOKEN=...").
//   (appelé par scripts/moodle-bootstrap.sh)
define('CLI_SCRIPT', true);
require('/opt/bitnami/moodle/config.php');
require_once($CFG->dirroot.'/course/lib.php');
require_once($CFG->dirroot.'/user/lib.php');
require_once($CFG->dirroot.'/lib/externallib.php');
require_once($CFG->dirroot.'/lib/enrollib.php');
global $DB, $CFG;

// ── 1. Activer les Web Services + protocole REST ────────────────────────────
set_config('enablewebservices', 1);
// Service mobile : permet à n'importe quel utilisateur (élève) d'obtenir un token via
// login/token.php (validation des identifiants pour le "login via Moodle").
set_config('enablemobilewebservice', 1);
$protocols = (string)get_config('core', 'webserviceprotocols');
$list = array_filter(array_map('trim', explode(',', $protocols)));
if (!in_array('rest', $list)) { $list[] = 'rest'; }
set_config('webserviceprotocols', implode(',', $list));

// ── 2. Service externe dédié "cpm_service" + ses fonctions ──────────────────
$functions = [
    'core_webservice_get_site_info',
    'core_course_get_courses',
    'core_course_get_courses_by_field',
    'core_enrol_get_enrolled_users',
    'core_enrol_get_users_courses',
    'core_user_get_users',
    'core_user_get_users_by_field',
    'enrol_manual_enrol_users',
    'core_course_create_courses',
    'core_user_create_users',
    'mod_assign_get_assignments',
    'mod_assign_save_grade',
    'gradereport_user_get_grade_items',
];
$service = $DB->get_record('external_services', ['shortname' => 'cpm_service']);
if (!$service) {
    $service = (object)[
        'name' => 'CloudPoolManager', 'shortname' => 'cpm_service', 'enabled' => 1,
        'restrictedusers' => 0, 'downloadfiles' => 1, 'uploadfiles' => 1,
        'timecreated' => time(), 'timemodified' => time(),
    ];
    $service->id = $DB->insert_record('external_services', $service);
} else {
    $DB->set_field('external_services', 'enabled', 1, ['id' => $service->id]);
}
foreach ($functions as $fn) {
    $exists = $DB->record_exists('external_services_functions',
        ['externalserviceid' => $service->id, 'functionname' => $fn]);
    if (!$exists) {
        $DB->insert_record('external_services_functions',
            (object)['externalserviceid' => $service->id, 'functionname' => $fn]);
    }
}

// ── 3. Token permanent pour l'admin sur ce service ──────────────────────────
$admin = get_admin();
$context = context_system::instance();
$existing = $DB->get_record('external_tokens',
    ['externalserviceid' => $service->id, 'userid' => $admin->id, 'tokentype' => EXTERNAL_TOKEN_PERMANENT]);
if ($existing) {
    $token = $existing->token;
} else if (class_exists('\core_external\util') && method_exists('\core_external\util', 'generate_token')) {
    $token = \core_external\util::generate_token(
        EXTERNAL_TOKEN_PERMANENT, $service, $admin->id, $context, 0, '');
} else {
    $token = external_generate_token(
        EXTERNAL_TOKEN_PERMANENT, $service, $admin->id, $context, 0, '');
}

// ── 4. Cours de démo ────────────────────────────────────────────────────────
function ensure_course($shortname, $fullname) {
    global $DB;
    $c = $DB->get_record('course', ['shortname' => $shortname]);
    if ($c) { return $c; }
    $data = (object)[
        'category' => 1, 'fullname' => $fullname, 'shortname' => $shortname,
        'summary' => '', 'summaryformat' => FORMAT_HTML, 'format' => 'topics', 'visible' => 1,
    ];
    return create_course($data);
}
$courses = [
    ensure_course('CPM-PY101', 'Python 101 (démo CloudPoolManager)'),
    ensure_course('CPM-DS200', 'Data Science 200 (démo CloudPoolManager)'),
];

// ── 5. Utilisateurs de démo ─────────────────────────────────────────────────
function ensure_user($username, $first, $last, $email) {
    global $DB, $CFG;
    $u = $DB->get_record('user', ['username' => $username]);
    if ($u) { return $u; }
    $user = (object)[
        'username' => $username, 'auth' => 'manual', 'confirmed' => 1,
        'mnethostid' => $CFG->mnet_localhost_id,
        'firstname' => $first, 'lastname' => $last, 'email' => $email,
        'password' => 'Student_2026!',
    ];
    $id = user_create_user($user, true, false);
    return $DB->get_record('user', ['id' => $id]);
}
$students = [
    ensure_user('alice',   'Alice',   'Martin',  'alice@example.com'),
    ensure_user('bob',     'Bob',     'Durand',  'bob@example.com'),
    ensure_user('charlie', 'Charlie', 'Bernard', 'charlie@example.com'),
    ensure_user('diana',   'Diana',   'Petit',   'diana@example.com'),
];
$teacher = ensure_user('prof1', 'Paul', 'Prof', 'prof1@example.com');

// ── 6. Inscriptions (manual enrol) ──────────────────────────────────────────
function enrol_in($courseid, $userid, $roleshortname) {
    global $DB;
    $role = $DB->get_record('role', ['shortname' => $roleshortname]);
    if (!$role) { return; }
    $plugin = enrol_get_plugin('manual');
    $instance = $DB->get_record('enrol', ['courseid' => $courseid, 'enrol' => 'manual']);
    if (!$instance) {
        $course = $DB->get_record('course', ['id' => $courseid]);
        $plugin->add_default_instance($course);
        $instance = $DB->get_record('enrol', ['courseid' => $courseid, 'enrol' => 'manual']);
    }
    $plugin->enrol_user($instance, $userid, $role->id);
}
foreach ($courses as $c) {
    foreach ($students as $s) { enrol_in($c->id, $s->id, 'student'); }
    enrol_in($c->id, $teacher->id, 'editingteacher');
}

// ── 7. S'assurer que le service mobile est activé (login/token.php des élèves) ──
$DB->set_field('external_services', 'enabled', 1, ['shortname' => 'moodle_mobile_app']);
purge_all_caches();

// ── Résumé machine-lisible ──────────────────────────────────────────────────
echo "TOKEN=$token\n";
foreach ($courses as $c) { echo "COURSE id={$c->id} shortname={$c->shortname}\n"; }
echo "STUDENTS=" . implode(',', array_map(fn($s) => $s->email, $students)) . "\n";
echo "TEACHER=prof1@example.com (mot de passe Student_2026!)\n";
echo "OK\n";
