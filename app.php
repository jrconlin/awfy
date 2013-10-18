<?php
// Main File.

//    In an ideal world, these would be useful. They're not.
//    header("Access-Control-Allow-Origin: *");
//    header("Access-Control-Allow-Methods: GET,POST,OPTIONS");
//    header("Access-Control-Allow-Headers: Authorization,Content-Type,Accept,Origin,User-Agent,DNT,Cache-Control,Keep-Alive,X-Requested-With,If-Modified-Since");

    include_once('vars.php');
    $days = 0;
    $prevDays = 0;
    $days = floor((time() - filectime($dayFile))/60);
    $dayInfo = Array('count'=>0);

    // This could use SQL db cleverness, but I don't.
    // less moving parts is better.
    try {
        // Append the event to the Day Log file.
        if (file_exists($dayFile)) {
            file_put_contents($dayLog,
                file_get_contents($dayFile), FILE_APPEND|LOCK_EX);
        }
        // Get the current info from the local file.
        $dayInfo = json_decode(file_get_contents($dayFile),True);
    } catch (Exception $e) {
        error("An exception happened! ". print_r($e, true));
    }

    // Someone hit the button!
    if (array_key_exists('reset', $_POST)) {
        $remote_addr = $_SERVER['REMOTE_ADDR'];
        file_put_contents($dayFile, json_encode(array('prev' => $days,
                                       'count' => $dayInfo['count']+1,
                                       'resetter' => $remote_addr)));
        $fsize = filesize($dayLog);
        $prevDays = $days;
        $days = 0;
        // Send out the announcements.
        $ch = curl_init();
        curl_setopt($ch, CURLOPT_URL, 'http://'.$host.'/alert.php');
        curl_setopt($ch, CURLOPT_POSTFIELDS, "s=$fsize");
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, TRUE);
        curl_exec($ch);
    }
    else {
        // otherwise just display what we've recorded earlier.
        $dayInfo = json_decode(file_get_contents($dayFile));
        $prevDays = $dayInfo->prev;
    }

?><!DOCTYPE HTML>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width">
<meta name="viewport" content="initial-scale=1.0">
<title>Are We Fired Yet?</title>
<link href='http://fonts.googleapis.com/css?family=Nothing+You+Could+Do&v2' rel='stylesheet' type='text/css'>
<link rel="stylesheet" href="style.css" type="text/css" />
<meta http-equiv="refresh" content="60;url=http://<?php print $host ?>" />
<link rel="icon" href="img/hr_icon.png" type="image/x-icon" />
<link rel="apple-touch-icon" href="img/hr_icon_128.png" />
<meta name="apple-mobile-web-app-capable" content="yes" />
<meta name="viewport" content="width = device-width" />
<meta name="viewport" content="initial-scale = 1.0" />
<meta property="og:url" content="http://evilonastick.com/hr/" />
<meta property="og:image" content="http://<?php print $host?>/img/hr_icon_128.png" />
<meta property="og:title" content="Are we fired yet?" />
<meta property="og:type" content="website" />
</head>
<body>
<div class="sign">
<h1>It has been
<div class="days"><?php print $days; ?></div>
<span class="period">minutes</span> since the last<br/> HR Violation.</h1>
<h2>Previous period was <?php print $prevDays; ?> <span class="psmall period ">minutes</span></h2>
<h3>Keep up the good work!</h3>
</div>
<form action="index<?php print $offset;?>.php" method="POST"><input type="hidden" name="reset" value="y"><input type="submit" value="reset" class="r reset" /></form>
<div id="webapp">
<div id="pushctl"><b>Neato! You can do push!</b><br>Tap me to know when resets happen</div>
<button id="install">Tap me to install as an app.</button>
</div>
<div id="bucket" style="display:none"></div>
<script lang="javascript">
    // semi-hack to set the Javascript host from the vars file.
    var HOST = "<?php print $host; ?>";
</script>
<script lang="javascript" src="cookie.js"></script>
<script lang="javascript" src="app1.js"></script>
</body>
<span class="debug">
<?php
?>
<span>
</html>
