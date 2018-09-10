//
//  ConversationViewController.m
//  KeybaseShare
//
//  Created by Michael Maxim on 8/31/18.
//  Copyright © 2018 Keybase. All rights reserved.
//

#import "ConversationViewController.h"
#import "keybase/keybase.h"
#import "Fs.h"

#if TARGET_OS_SIMULATOR
const BOOL isSimulator = YES;
#else
const BOOL isSimulator = NO;
#endif

@interface ConversationViewController ()
@property UISearchController* searchController;
@property NSArray* unfilteredInboxItems;
@property NSArray* filteredInboxItems;
@end

@implementation ConversationViewController

- (void)viewDidLoad {
  [super viewDidLoad];
  
  self.preferredContentSize = CGSizeMake(self.view.frame.size.width, 2*self.view.frame.size.height);
  self.searchController = [[UISearchController alloc] initWithSearchResultsController:nil];
  self.searchController.searchResultsUpdater = self;
  self.searchController.hidesNavigationBarDuringPresentation = false;
  self.searchController.dimsBackgroundDuringPresentation = false;
  self.definesPresentationContext = YES;
  [self.tableView setTableHeaderView:self.searchController.searchBar];
  
  UIActivityIndicatorView* av = [[UIActivityIndicatorView alloc] initWithActivityIndicatorStyle:UIActivityIndicatorViewStyleGray];
  [self.view addSubview:av];
  [av setTranslatesAutoresizingMaskIntoConstraints:NO];
  [av setHidesWhenStopped:YES];
  [av bringSubviewToFront:self.view];
  [av startAnimating];
  [self.tableView addConstraints:@[
     [NSLayoutConstraint constraintWithItem:av
                                  attribute:NSLayoutAttributeCenterX
                                  relatedBy:NSLayoutRelationEqual
                                     toItem:self.tableView
                                  attribute:NSLayoutAttributeCenterX
                                 multiplier:1 constant:0],
     [NSLayoutConstraint constraintWithItem:av
                                  attribute:NSLayoutAttributeCenterY
                                  relatedBy:NSLayoutRelationEqual
                                     toItem:self.tableView
                                  attribute:NSLayoutAttributeCenterY
                                 multiplier:1 constant:0]
     ]
   ];
  
  dispatch_async(dispatch_get_main_queue(), ^{
    NSError* error = NULL;
    NSDictionary* fsPaths = [[FsHelper alloc] setupFs:YES setupSharedHome:NO];
    KeybaseExtensionInit(fsPaths[@"home"], fsPaths[@"sharedHome"], fsPaths[@"logFile"], @"prod", isSimulator, NULL, NULL, &error);
    if (error != nil) {
      NSLog(@"Failed to init: %@", error);
      return;
    }
   
    [self setUnfilteredInboxItems:[NSArray new]];
    [self setFilteredInboxItems:[NSArray new]];
    NSString* jsonInbox = KeybaseExtensionGetInbox(&error);
    if (jsonInbox == nil) {
      NSLog(@"failed to get inbox: %@", error);
    } else {
      [self parseInbox:jsonInbox];
    }
    [av stopAnimating];
    [self.tableView reloadData];
  });
}

- (void)parseInbox:(NSString*)jsonInbox {
  NSError *error = nil;
  NSData *data = [jsonInbox dataUsingEncoding:NSUTF8StringEncoding];
  NSArray *items = [NSJSONSerialization JSONObjectWithData:data options: NSJSONReadingMutableContainers error: &error];
  if (!items) {
    NSLog(@"parseInbox: error parsing JSON: %@", error);
  } else {
    [self setUnfilteredInboxItems:items];
    [self setFilteredInboxItems:items];
  }
}

- (void)didReceiveMemoryWarning {
  KeybaseExtensionForceGC();
  [super didReceiveMemoryWarning];
}

#pragma mark - Table view data source

- (NSInteger)numberOfSectionsInTableView:(UITableView *)tableView {
    return 1;
}

- (NSInteger)tableView:(UITableView *)tableView numberOfRowsInSection:(NSInteger)section {
    return [self.filteredInboxItems count];
}

- (NSDictionary*)getItemAtIndex:(NSIndexPath*)indexPath {
  NSInteger index = [indexPath item];
  return self.filteredInboxItems[index];
}

- (UITableViewCell *)tableView:(UITableView *)tableView cellForRowAtIndexPath:(NSIndexPath *)indexPath {
  UITableViewCell *cell = [tableView dequeueReusableCellWithIdentifier:@"ConvCell"];
  if (NULL == cell) {
    cell = [[UITableViewCell alloc]  initWithStyle:UITableViewCellStyleDefault reuseIdentifier:@"ConvCell"];
  }
  NSDictionary* item = [self getItemAtIndex:indexPath];
  [[cell textLabel] setText:item[@"Name"]];
  return cell;
}

- (void)tableView:(UITableView *)tableView didSelectRowAtIndexPath:(NSIndexPath *)indexPath {
  NSDictionary* conv = [self getItemAtIndex:indexPath];
  if (self.delegate) {
    [self.delegate convSelected:conv];
  }
}

- (void)updateSearchResultsForSearchController:(UISearchController *)searchController {
  NSString* term = [searchController.searchBar.text lowercaseString];
  if ([term length] == 0) {
    [self setFilteredInboxItems:self.unfilteredInboxItems];
  } else {
    NSPredicate* pred = [NSPredicate predicateWithBlock:^BOOL(id obj, NSDictionary* bindings) {
      NSDictionary* item = obj;
      return [item[@"Name"] containsString:term];
    }];
    NSArray* filtered = [self.unfilteredInboxItems filteredArrayUsingPredicate:pred];
    [self setFilteredInboxItems:filtered];
  }
  [self.tableView reloadData];
}

@end
