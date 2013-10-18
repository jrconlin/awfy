<?php

// Register the new Endpoint.

include_once('vars.php');

// run the gauntlet.
    if (empty($_POST["endpoint"])) {
    error("No endpoint");
            exit();
    }
    $endpoint = trim($_POST['endpoint']);
    if ($n = strpos($endpoint,'@') !== false) {
        error("bad endpoint $n");
        exit();
    }
    if ($n = strpos($endpoint,
            'https://updates.push.services.mozilla.com/update/') !== 0) {
        error("Invalid endpoint");
        exit();
    }
    print_r("n: $n\n");

    // append the endpoint to the endpoints file.
    // Yes, in a real world environment, i'd use a database.
    $data = array();
    if (file_exists($registry)) {

        $data = file($registry, FILE_SKIP_EMPTY_LINES);
    }

    // in_array was being a pain, so doing this the old way.
    $found = false;
    foreach($data as $item) {
        if (trim($item) == $endpoint) {
            $found = true;
        }
    }
    if (! $found) {
        $data[] = $endpoint;
    }

    // likewise writing the join'd array was embedding lots of \n's
    $file = fopen($registry, "w");
    foreach ($data as $line) {
        if (!empty($line) && strlen($line) > 0) {
            fwrite($file, trim($line)."\n");
        }
    }
    fclose($file);
?>
