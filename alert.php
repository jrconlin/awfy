<?php

// Send out the alerts.
//
    include_once('vars.php');

    try {
        if(intval($_POST['s']) != filesize($dayLog)) {
            exit;
        }
    } catch (Exception $e){
        error($e);
        exit;
    }

    $urls = file($registry);
    // Use PHP5.5's nifty multi curl ability to broadcast.
    // (I'd love for this to be backgrounded, but that's not PHP's way.)

    $curlm = curl_multi_init();
    $channels = array();
    foreach ($urls as $url) {
        $curl = curl_init();
        curl_setopt($curl, CURLOPT_URL, $url);
        curl_setopt($curl, CURLOPT_CUSTOMREQUEST, "PUT");
        curl_setopt($curl, CURLOPT_POSTFIELDS, "version=".time());
        // will use current time version= is not specified.
        curl_multi_add_handle($curlm, $curl);
    }
    $active = null;
    do {
        $mrc = curl_multi_exec($curlm, $active);
    } while ($mrc == CURLM_CALL_MULTI_PERFORM);

    while ($active && $mrc == CURLM_OK) {
        if (curl_multi_select($curlm) != -1) {
            do {
                $mrc = curl_multi_exec($curlm, $active);
            } while ($mrc == CURLM_CALL_MULTI_PERFORM);
        }
    }

    // I don't care about the resuts.
    foreach ($channels as $channel) {
        echo curl_multi_remove_handle($multi, $channel);
    }

    curl_multi_close($curlm);

?>
